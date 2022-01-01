package main

import (
	"compress/gzip"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/kiteco/kiteco/kite-go/lang/python/answers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/seo"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`

var sitemapRoot, docsRoot, examplesRoot, answersRoot url.URL

func init() {
	var err error
	var url *url.URL

	url, err = url.Parse("https://www.kite.com/")
	if err != nil {
		panic(err)
	}
	sitemapRoot = *url

	url, err = url.Parse("https://www.kite.com/python/docs")
	if err != nil {
		panic(err)
	}
	docsRoot = *url

	url, err = url.Parse("https://www.kite.com/python/examples")
	if err != nil {
		panic(err)
	}
	examplesRoot = *url

	url, err = url.Parse("https://www.kite.com/python/answers")
	if err != nil {
		panic(err)
	}
	answersRoot = *url
}

func main() {
	var generate bool
	var upload bool
	var isProd bool
	var sitemapLocalDir string
	var bucket string

	flag.BoolVar(&generate, "generate", false, "generate sitemaps")
	flag.BoolVar(&upload, "upload", false, "upload sitemaps")
	flag.BoolVar(&isProd, "prod", false, "if sitemaps should be uploaded for production use")
	flag.StringVar(&sitemapLocalDir, "dir", "", "sitemap directory")
	flag.StringVar(&bucket, "bucket", "kite-data", "sitemap s3 upload bucket")
	flag.Parse()

	if sitemapLocalDir == "" {
		log.Println("You need to input a dir to which generated sitemaps can be written, or from which sitemaps can be uploaded")
		return
	}
	switch {
	case generate && upload:
		err := generateSitemaps(sitemapLocalDir)
		if err != nil {
			log.Println("err in generate: ", err)
			return
		}
		err = uploadSitemaps(sitemapLocalDir, bucket, isProd)
		if err != nil {
			log.Println("err in upload: ", err)
		}

	case generate:
		err := generateSitemaps(sitemapLocalDir)
		if err != nil {
			log.Println("err in generate: ", err)
		}
	case upload:
		err := uploadSitemaps(sitemapLocalDir, bucket, isProd)
		if err != nil {
			log.Println("err in upload: ", err)
		}
	}
}

func generateSitemaps(sitemapDir string) error {
	smi := newSitemapIndex()

	answersURLs, err := addAnswers(make(urlSet))
	if err != nil {
		return err
	}
	smi = smi.Append(sitemapRoot, "sitemap_answers_%d.xml.gz", answersURLs)

	docsURLs, err := addDocs(make(urlSet))
	if err != nil {
		return err
	}
	smi = smi.Append(sitemapRoot, "sitemap_docs_%d.xml.gz", docsURLs)

	examplesURLs, err := addExamples(make(urlSet))
	if err != nil {
		return err
	}
	smi = smi.Append(sitemapRoot, "sitemap_examples_%d.xml.gz", examplesURLs)

	os.MkdirAll(sitemapDir, 0755)
	for fname, sm := range smi.Sitemaps {
		if err := writeXMLGZ(filepath.Join(sitemapDir, fname), sm); err != nil {
			return err
		}
	}
	if err := writeXMLGZ(filepath.Join(sitemapDir, "sitemap-index.xml.gz"), smi); err != nil {
		return err
	}
	return nil
}

func writeXMLGZ(path string, data interface{}) (err error) {
	xmlgzW, err := os.Create(path)
	if err != nil {
		return err
	}
	defer errors.Defer(&err, xmlgzW.Close)
	xmlW := gzip.NewWriter(xmlgzW)
	defer errors.Defer(&err, xmlW.Close)
	if _, err := xmlW.Write([]byte(xmlHeader)); err != nil {
		return err
	}
	if err := xml.NewEncoder(xmlW).Encode(data); err != nil {
		return err
	}
	return nil
}

func addAnswers(urls urlSet) (urlSet, error) {
	idx, err := answers.Load(answers.DefaultPath)
	if err != nil {
		return urls, err
	}
	for slug := range idx.Slugs {
		url := answersRoot
		url.Path = path.Join(url.Path, slug)
		urls[url.String()] = struct{}{}
	}
	return urls, nil
}

func addDocs(urls urlSet) (urlSet, error) {
	data, err := seo.Load(seo.DefaultDataPath)
	if err != nil {
		return urls, err
	}
	data.IterateCanonicalLinkPaths(func(dpath pythonimports.DottedPath) bool {
		docsURL := docsRoot
		docsURL.Path = path.Join(docsURL.Path, dpath.String())
		urls[docsURL.String()] = struct{}{}
		return true
	})
	return urls, nil
}

func addExamples(urls urlSet) (urlSet, error) {
	importGraph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		return urls, errors.Errorf("error creating new import graph: %v", err)
	}
	curatedSearcher, err := pythoncuration.NewSearcher(importGraph, &pythoncuration.DefaultSearchOptions)
	if err != nil {
		return urls, errors.Errorf("err creating new curated searcher: %v", err)
	}
	curatedMap := curatedSearcher.AllCurated()

	//hack due to how '%' is handled by React Router
	replacer := strings.NewReplacer(" ", "-", "%", "percent-sign")
	for id, snippet := range curatedMap {
		title := url.PathEscape(strings.ToLower(replacer.Replace(snippet.Curated.Snippet.Title)))
		pkg := strings.ToLower(snippet.Curated.Snippet.Package)
		examplesURL := examplesRoot
		examplesURL.Path = path.Join(examplesURL.Path, strconv.FormatInt(id, 10), pkg+"-"+title)
		urls[examplesURL.String()] = struct{}{}
	}
	return urls, nil
}

func uploadSitemaps(sitemapDir string, bucket string, isProd bool) error {
	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return err
	}
	dateString := time.Now().Format(time.UnixDate)

	s3bucket := s3.New(auth, aws.USWest).Bucket(bucket)
	fileInfos, err := ioutil.ReadDir(sitemapDir)
	if err != nil {
		return fmt.Errorf("error reading sitemap dir: %s -> %v", sitemapDir, err)
	}
	for _, info := range fileInfos {
		name := info.Name()
		if strings.HasPrefix(name, "sitemap") && strings.HasSuffix(name, ".xml.gz") {
			//open file
			file, err := os.Open(filepath.Join(sitemapDir, name))
			if err != nil {
				return fmt.Errorf("error opening sitemap path %s -> %v", name, err)
			}
			var s3Path string
			if isProd {
				s3Path = name
			} else {
				s3Path = fmt.Sprintf("%s%s%s%s", "sitemaps/", dateString, "/", name)
			}
			if err := s3bucket.PutReader(s3Path, file, info.Size(), "application/xml", s3.PublicRead, s3.Options{
				ContentEncoding: "gzip",
			}); err != nil {
				return fmt.Errorf("error Putting file %s to s3 -> %v", name, err)
			}
		}
	}
	fmt.Println("success in uploading sitemaps!")
	return nil
}
