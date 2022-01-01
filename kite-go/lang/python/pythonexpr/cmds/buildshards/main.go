package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func main() {
	var (
		rundbRoot string
		output    string
	)

	flag.StringVar(&rundbRoot, "rundbRoot", "", "rundb root of pipeline run")
	flag.StringVar(&output, "output", "shards.json", "where to write shard definition")
	flag.Parse()

	rundbShards, err := findShards(rundbRoot)
	if err != nil {
		log.Fatalln(err)
	}

	if len(rundbShards) == 0 {
		log.Fatalln("found no shards in", rundbRoot)
	}

	var shards []pythonexpr.Shard
	for _, shard := range rundbShards {
		pkglist, err := readPackageList(shard.packagelistPath)
		if err != nil {
			log.Fatalf("error reading packagelist %s: %s", shard.packagelistPath, err)
		}
		shards = append(shards, pythonexpr.Shard{
			Packages:  pkglist,
			ModelPath: shard.modelPath,
		})
	}

	buf, err := json.MarshalIndent(shards, "", "  ")
	if err != nil {
		log.Fatalln(err)
	}

	err = ioutil.WriteFile(output, buf, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
}

func readPackageList(fn string) (pythonexpr.PackageList, error) {
	f, err := fileutil.NewReader(fn)
	if err != nil {
		return pythonexpr.PackageList{}, err
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return pythonexpr.PackageList{}, err
	}

	lines := strings.Split(string(buf), "\n")

	var pkglist pythonexpr.PackageList
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 {
			pkglist = append(pkglist, trimmed)
		}
	}

	sort.Strings(pkglist)

	return pkglist, nil
}

type shard struct {
	modelPath       string
	packagelistPath string
}

func findShards(path string) ([]shard, error) {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	s3url, err := awsutil.ValidateURI(path)
	if err != nil {
		return nil, err
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := s3.New(sess, defaults.Get().Config)

	listReq := &s3.ListObjectsInput{
		Bucket:    aws.String(s3url.Host),
		Prefix:    aws.String(strings.TrimPrefix(s3url.Path, "/")),
		Delimiter: aws.String("/"),
	}

	listResp, err := svc.ListObjects(listReq)
	if err != nil {
		return nil, err
	}

	b, _ := json.MarshalIndent(listResp, "", "  ")
	fmt.Println(string(b))

	var candidates []shard
	for _, prefix := range listResp.CommonPrefixes {
		if strings.Contains(*prefix.Prefix, "expr-") {
			candidates = append(candidates, shard{
				modelPath:       fileutil.Join("s3://", s3url.Host, *prefix.Prefix, "serve"),
				packagelistPath: fileutil.Join("s3://", s3url.Host, *prefix.Prefix, "packagelist.txt"),
			})
		}
	}

	log.Println(candidates)

	// Hacky "validation" of the discovery phase above
	var shards []shard
	for _, shard := range candidates {
		exists, err := awsutil.Exists(fileutil.Join(shard.modelPath, "expr_model.frozen.pb"))
		if err != nil || !exists {
			continue
		}
		exists, err = awsutil.Exists(shard.packagelistPath)
		if err != nil || !exists {
			continue
		}
		shards = append(shards, shard)
	}

	return shards, nil
}
