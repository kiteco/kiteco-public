package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/goamz/goamz/aws"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"

	_ "github.com/lib/pq"
)

var (
	largeFileThreshold = int(1e5) // files over this size (in bytes) may be noise
	nRetriesFilesDB    = 3
	nRetriesFetchS3    = 3

	regions = map[string]struct{}{
		"us-west-1": struct{}{},
	}

	uriRegEx = regexp.MustCompile(`export LOCALFILES_DB_URI="(?P<uri>.+)"`)
)

func getFilesDBURI(region, secrets string) string {
	path := "s3://" + filepath.Join(secrets, "config", region+".sh")
	rdr, err := awsutil.NewS3Reader(path)
	if err != nil {
		log.Fatalf("error getting secrets reader for %s: %v \n", region, err)
	}
	defer rdr.Close()

	buf, err := ioutil.ReadAll(rdr)
	if err != nil {
		log.Fatalf("error reading secrets for for %s: %v\n", region, err)
	}

	line := uriRegEx.FindString(string(buf))
	uri := strings.TrimPrefix(line, `export LOCALFILES_DB_URI="`)
	uri = strings.TrimSuffix(uri, `"`)
	return uri
}

var excludePatterns = []*regexp.Regexp{
	// these are temporaries created by setuptools
	regexp.MustCompile(`/build/lib/`),
	regexp.MustCompile(`/build/.?dist\..*/`), // matches e.g. ".../build/bdist.macosx-10.5-x86_64/..."
	regexp.MustCompile(`/build/scripts.*/`),  // matches .e.g ".../build/scripts-2.7/..."

	// these directories contain code for user's applications, usually not run directly by user
	regexp.MustCompile(`/Library/Application Support/`),
}

func filterFiles(all []*localfiles.File) []*localfiles.File {
	var filtered []*localfiles.File
	for _, f := range all {
		var ignore bool
		for _, pat := range excludePatterns {
			if pat.MatchString(f.Name) {
				ignore = true
				break
			}
		}

		if !ignore {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func main() {
	var args struct {
		IDs     []string `arg:"positional,required,help:list of user IDs"`
		Corpus  string   `arg:"required"`
		Verbose bool
	}
	arg.MustParse(&args)

	// s3 bucket containing secrets
	secrets := envutil.MustGetenv("SECRETS_BUCKET")

	// local code stores for all regions
	stores := make(map[string]*localfiles.ContentStore)

	for region := range regions {
		fdb := localfiles.FileDB("postgres", getFilesDBURI(region, secrets))
		opts := localfiles.DefaultContentStoreOptions
		opts.BucketName = "kite-local-content"
		if region != "us-west-1" {
			opts.BucketName = "kite-local-content-" + region
		}
		opts.Region = aws.Regions[region]

		if err := fdb.DB().Ping(); err != nil {
			log.Fatalf("unable to ping db in %s: %v\n", region, err)
		}

		store, err := localfiles.NewContentStore(opts, fdb)
		if err != nil {
			log.Fatalf("error getting content store for %s: %v\n", region, err)
		}
		stores[region] = store
	}

	var noMachineIDs, noFiles []string

	for _, uidStr := range args.IDs {
		start := time.Now()

		// parse user ID
		uid, err := strconv.ParseInt(strings.TrimSpace(uidStr), 10, 64)
		if err != nil {
			log.Fatalf("unable to parse userID from `%s`: %v\n", uidStr, err)
		}

		// get user's machine IDs
		var mids []string
		var region string
		var store *localfiles.ContentStore
	GetMachines:
		for i := 0; i < nRetriesFilesDB; i++ {
			for r, s := range stores {
				mids, err = s.Files.Machines(uid)
				if err == nil && len(mids) > 0 {
					region = r
					store = s
					break GetMachines
				}
				if args.Verbose {
					log.Printf("error getting machine IDs for %d in %s: %v\n", uid, r, err)
				}
			}
		}

		if len(mids) == 0 || err != nil {
			log.Printf("error getting machine IDs for %d: %v, skipping\n", uid, err)
			noMachineIDs = append(noMachineIDs, uidStr)
			continue
		}

		// get user's files for each machine
		for _, mid := range mids {
			// fetch "file headers"
			var files []*localfiles.File
			for i := 0; i < nRetriesFilesDB; i++ {
				files, err = store.Files.List(uid, mid, "", ".py")
				if len(files) > 0 && err == nil {
					break
				}
			}

			if len(files) == 0 || err != nil {
				log.Printf("unable to find files for user %v and machine %s in %s: %v, skipping\n", uid, mid, region, err)
				noFiles = append(noFiles, fmt.Sprintf("%s:%d:%s", region, uid, mid))
				continue
			}

			var nonAbsPath, tooLarge, unableToFetch, alreadyDownloaded, wrote int
			filtered := filterFiles(files)
			for _, f := range filtered {
				if !path.IsAbs(f.Name) {
					log.Printf("non absolute path for %s:%d:%s/%s, skipping \n", region, uid, mid, f.Name)
					nonAbsPath++
					continue
				}

				newPath := path.Join(args.Corpus, fmt.Sprintf("%d", uid), mid, f.Name)
				if _, err := os.Stat(newPath); err == nil {
					alreadyDownloaded++
					continue
				}

				var content []byte
				var err error
				for i := 0; i < nRetriesFetchS3; i++ {
					content, err = store.Get(f.HashedContent)
					if err == nil {
						break
					}
				}

				if err != nil {
					unableToFetch++
					log.Printf("error getting content %s:%d:%s/%s: %v, skipping\n", region, uid, mid, f.Name, err)
					continue
				}

				if len(content) > pathSelectionOpts.LargeFileThreshold {
					tooLarge++
					continue
				}

				if err := os.MkdirAll(filepath.Dir(newPath), 0777); err != nil {
					log.Fatalf("error creating dirs for `%s`: %v\n", filepath.Dir(newPath), err)
				}
				if err := ioutil.WriteFile(newPath, content, 0777); err != nil {
					log.Fatalf("error writing `%s`: %v\n", newPath, err)
				}
				wrote++
			}

			fmt.Printf("for %s:%d:%s, wrote %d files (%d already downloaded): ",
				region, uid, mid, wrote, alreadyDownloaded)
			fmt.Printf("%d had non absolute paths, %d were too large, failed to fetch %d from s3\n",
				nonAbsPath, tooLarge, unableToFetch)
		}

		fmt.Printf("for %s:%d, took %v\n", region, uid, time.Since(start))
	}

	fmt.Println("Failed to get machine ids for users:")
	for _, uid := range noMachineIDs {
		fmt.Println("  ", uid)
	}

	fmt.Println("Failed to get files for:")
	for _, id := range noFiles {
		fmt.Println("  ", id)
	}
}
