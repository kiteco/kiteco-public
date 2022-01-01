package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	humanize "github.com/dustin/go-humanize"
	"github.com/mholt/archiver"
)

const (
	dir    = "2019-03"
	bucket = "github-crawl-kite"
)

type fetchType string

var (
	fetchViaHTTPSGet = fetchType("get")
)

func markFetched(hostPort, id string) {
	url := fmt.Sprintf("http://%s/fetched?ids=%s", hostPort, id)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("update fetched error:", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
}

func markErrored(hostPort, id string) {
	url := fmt.Sprintf("http://%s/errored?ids=%s", hostPort, id)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("update errored error:", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
}

func crawl(n int, hostPort, outputdir string) {
	log.Printf("starting crawl... (concurrency=%d)", n)

	entryChan := make(chan *repoEntry)

	for i := 0; i < n; i++ {
		go func() {
			for entry := range entryChan {
				owner := entry.Owner
				repo := entry.Repo
				logf := func(msg string, objs ...interface{}) {
					log.Printf("[%s/%s] %s", owner, repo, fmt.Sprintf(msg, objs...))
				}

				fn, err := fetch(fetchViaHTTPSGet, entry, outputdir)
				if err != nil {
					markErrored(hostPort, fmt.Sprintf("%d", entry.ID))
					logf("clone error:", err)
					continue
				}
				err = upload(bucket, dir, fn)
				if err != nil {
					markErrored(hostPort, fmt.Sprintf("%d", entry.ID))
					logf("upload error:", err)
					continue
				}
				markFetched(hostPort, fmt.Sprintf("%d", entry.ID))
				logf("uploaded")

				err = os.RemoveAll(fn)
				if err != nil {
					logf("error removing %s: %s", fn, err)
				}
			}
		}()
	}

	for {
		var err error
		var entries []*repoEntry

		err = func() error {
			url := fmt.Sprintf("http://%s/next-repos?n=%d", hostPort, n*2)
			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("error fetching entries, retrying in 5s: %s", err)
			}

			defer resp.Body.Close()

			err = json.NewDecoder(resp.Body).Decode(&entries)
			if err != nil {
				return fmt.Errorf("unable to decode entries, retrying in 5s: %s", err)
			}

			return nil
		}()

		if err != nil {
			log.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, entry := range entries {
			entryChan <- entry
		}
	}
}

func fetch(method fetchType, entry *repoEntry, outputdir string) (string, error) {
	owner := entry.Owner
	repo := entry.Repo
	logf := func(msg string, objs ...interface{}) {
		log.Printf("[%s/%s] %s", owner, repo, fmt.Sprintf(msg, objs...))
	}

	tarfn := fmt.Sprintf("github.com__%d__%s__%s.tar.gz", entry.ID, owner, repo)
	destfn := path.Join(outputdir, tarfn)
	if _, err := os.Stat(destfn); err == nil || os.IsExist(err) {
		logf("already exists: %s", destfn)
		return destfn, nil
	}

	tempDir, err := ioutil.TempDir("", "github-crawler")
	if err != nil {
		log.Fatalln(err)
	}

	defer func() {
		os.RemoveAll(tempDir)
	}()

	var destDir string
	switch method {
	case fetchViaHTTPSGet:
		logf("https get")
		destDir, err = get(owner, repo, tempDir)
		if err != nil {
			return "", err
		}
	}

	logf("cleaning up unecessary files and directories")

	toremove := make(map[string]bool)

	// Remove .git and .github directories
	var removedBytes uint64
	toremove[path.Join(destDir, ".git")] = true
	toremove[path.Join(destDir, ".github")] = true

	err = filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return err
		}

		if info.Mode() == os.ModeSymlink {
			return err
		}

		// Remove non-text mimetypes
		mt := mime.TypeByExtension(filepath.Ext(path))
		if !strings.HasPrefix(mt, "text/") {
			toremove[path] = true
			removedBytes += uint64(info.Size())
			return nil
		}

		return err
	})
	if err != nil {
		return "", fmt.Errorf("walk: %s", err)
	}

	logf("removing %d unecessary files (%s)", len(toremove), humanize.Bytes(removedBytes))
	for r := range toremove {
		err = os.RemoveAll(r)
		if err != nil {
			return "", fmt.Errorf("remove: %s", err)
		}
	}

	logf("adding repo entry metadata")
	kiteMeta := filepath.Join(destDir, "kite-repo-entry.json")
	buf, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(kiteMeta, buf, os.ModePerm)
	if err != nil {
		return "", err
	}

	logf("creating tar archive %s", tarfn)

	tempfn := path.Join(tempDir, tarfn)
	err = archiver.NewTarGz().Archive([]string{destDir}, tempfn)
	if err != nil {
		return "", fmt.Errorf("tar.gz: %s", err)
	}

	logf("moving to %s", destfn)

	err = os.Rename(tempfn, destfn)
	if err != nil {
		return "", fmt.Errorf("mv: %s", err)
	}

	return destfn, nil
}

/*
func clone(owner, repo, tempDir string) (string, error) {
	cloneDir := path.Join(tempDir, owner, repo)
	err := os.MkdirAll(cloneDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("mkdir: %s", err)
	}

	_, err = git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:      fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		Depth:    1,
		Progress: ioutil.Discard,
	})
	if err != nil {
		return "", fmt.Errorf("plain clone: %s", err)
	}

	return cloneDir, nil
}
*/

func get(owner, repo, tempDir string) (string, error) {
	zipDest := path.Join(tempDir, "master.zip")

	out, err := os.Create(zipDest)
	if err != nil {
		return "", fmt.Errorf("create file: %s", err)
	}

	resp, err := http.Get(fmt.Sprintf("https://github.com/%s/%s/archive/master.zip", owner, repo))
	if err != nil {
		return "", fmt.Errorf("http get: %s", err)
	}

	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error downloading zip: %s", err)
	}

	err = out.Close()
	if err != nil {
		return "", fmt.Errorf("error closing: %s", err)
	}

	zipExtract := path.Join(tempDir, owner, repo)
	err = os.MkdirAll(zipExtract, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("mkdir: %s", err)
	}

	err = archiver.NewZip().Unarchive(zipDest, zipExtract)
	if err != nil {
		return "", fmt.Errorf("unzip: %s", err)
	}

	fis, err := ioutil.ReadDir(zipExtract)
	if err != nil {
		return "", fmt.Errorf("error listing %s: %s", zipExtract, err)
	}

	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), "-master") {
			trimmed := strings.TrimSuffix(fi.Name(), "-master")
			zipExtract = filepath.Join(zipExtract, fi.Name())
			zipExtractRename := filepath.Join(filepath.Dir(zipExtract), trimmed)
			err = os.Rename(zipExtract, zipExtractRename)
			if err != nil {
				return "", fmt.Errorf("error renaming %s -> %s: %s", zipExtract, zipExtractRename, err)
			}
			zipExtract = zipExtractRename
		}
	}

	return zipExtract, nil
}

func upload(bucket, dir, fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}

	defer f.Close()

	key := fmt.Sprintf("%s/%s", dir, filepath.Base(fn))
	putObjInput := &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   f,
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	s3client := s3.New(sess, aws.NewConfig())
	_, err = s3client.PutObject(putObjInput)
	if err != nil {
		return err
	}

	return nil
}
