package main

import (
	"bufio"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

const (
	rtdTemplate = "http://readthedocs.org/api/v1/project/%s/?format=json"
)

var (
	documentationRoot = "s3://kite-emr/datasets/documentation"

	// defaultPythonDocs points to a location that only contains the std library. Its used to filter out
	// packages from the target set.
	defaultPythonDocs = path.Join(documentationRoot, "python", "2015-07-20_14-03-02-PM", "python.json.gz")

	client *http.Client
)

func init() {
	// ReadTheDocs.org has a funky SSL setup, requiring
	// setting the max TLS version.
	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MaxVersion: tls.VersionTLS11,
			},
		},
	}
}

func main() {
	var (
		outputRoot string
		targetPkgs string
	)

	flag.StringVar(&outputRoot, "outputRoot", "artifacts/rtd-docs", "directory to download docs")
	flag.StringVar(&targetPkgs, "targetPkgs", "artifacts/target.txt", "list of target packages to download")
	flag.Parse()

	err := os.MkdirAll(outputRoot, os.ModePerm)
	if err != nil {
		log.Fatalln("could not create output directory:", err)
	}

	modules := loadModules(defaultPythonDocs)
	targetList := loadTargets(targetPkgs)

	for _, pkg := range targetList {
		if _, exists := modules[pkg]; exists {
			continue
		}
		if downloaded(pkg, outputRoot) {
			fmt.Println("====", "SKIPPING", pkg, "(already downloaded")
			continue
		}

		time.Sleep(time.Second * 1)

		fmt.Println("====", pkg)
		fmt.Println("\tdownloading rtd package info...")
		projectInfo, err := getRTDProjectInfo(pkg)
		if err != nil {
			log.Println(pkg, err)
			continue
		}

		fmt.Println("\tdownloading html zip of docs...")
		err = downloadHTMLZip(pkg, projectInfo.DownloadFormatURLs.HTMLZip, outputRoot)
		if err != nil {
			log.Println(pkg, err)
			continue
		}

		fmt.Fprintf(os.Stderr, "fetched %s\n", pkg)
	}
}

func loadTargets(path string) []string {
	in, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	r := bufio.NewScanner(in)

	var targets []string
	for r.Scan() {
		targets = append(targets, r.Text())
	}
	if err := r.Err(); err != nil {
		log.Fatal(err)
	}

	return targets
}

func loadModules(path string) pythondocs.Modules {
	s3r, err := awsutil.NewS3Reader(path)
	if err != nil {
		log.Fatal(err)
	}
	decomp, err := gzip.NewReader(s3r)
	if err != nil {
		log.Fatal(err)
	}
	dec := json.NewDecoder(decomp)

	modules := make(pythondocs.Modules)
	err = modules.Decode(dec)
	if err != nil {
		log.Fatal(err)
	}
	return modules
}

func makeRTDURL(pkg string) string {
	urlTemplate := fmt.Sprintf(rtdTemplate, strings.ToLower(pkg))
	parsed, err := url.Parse(urlTemplate)
	if err != nil {
		log.Fatal(err)
	}
	return parsed.String()
}

func getRTDProjectInfo(pkg string) (*pythondocs.RTDProjectInfo, error) {
	ep := makeRTDURL(pkg)
	resp, err := http.Get(ep)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(resp.Status)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var project pythondocs.RTDProjectInfo
	err = json.Unmarshal(buf, &project)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func downloaded(pkg, outputRoot string) bool {
	dest := path.Join(outputRoot, "rtd-docs", fmt.Sprintf("%s.zip", pkg))
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func downloadHTMLZip(pkg, htmlZip, outputRoot string) error {
	if htmlZip == "" {
		return fmt.Errorf("htmlZip url is empty")
	}

	zipURL, err := url.Parse(htmlZip)
	if err != nil {
		return err
	}

	switch zipURL.Scheme {
	case "http", "":
		zipURL.Scheme = "https"
	}

	resp, err := client.Get(zipURL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(resp.Status)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	dest := path.Join(outputRoot, "rtd-docs", fmt.Sprintf("%s.zip", pkg))
	err = ioutil.WriteFile(dest, buf, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
