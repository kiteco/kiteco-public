package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	googleAppengineURL = "https://cloud.google.com/appengine/docs/python/refdocs"
	logPrefix          = "[crawl-google]"
	logFlags           = log.LstdFlags | log.Lshortfile
	index              = "google.appengine.html"
)

var (
	outputDir string
)

func main() {
	flag.StringVar(&outputDir, "outputDir", "", "Directory to store crawled docs")
	flag.Parse()

	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	buf := fetchAndSave(index)

	// Extract modules from index
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(buf))
	if err != nil {
		log.Fatalf("error using goquery to parse response: %v", err)
	}
	links := doc.Find("a.reference.internal")
	if links.Length() == 0 {
		log.Fatalln("expected at least one link to modules/packages")
	}
	links.Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.HasSuffix(text, "package") || strings.HasSuffix(text, "module") {
			if url, ok := s.Attr("href"); ok && url != "#" {
				fetchAndSave(url)
			}
		}
	})
}

func fetchAndSave(name string) []byte {
	docURL := makeDocURL(name)
	result, err := http.Get(docURL)
	if err != nil {
		log.Fatalf("error getting URL: %v", err)
	}
	defer result.Body.Close()

	buf, err := ioutil.ReadAll(result.Body)
	if err != nil && err != io.EOF {
		log.Fatalf("error reading bytes from response body: %v", err)
	}

	if err := ioutil.WriteFile(filepath.Join(outputDir, name), buf, 0644); err != nil {
		log.Fatalf("error writing buf to file: %v\n", err)
	}

	return buf
}

func makeDocURL(name string) string {
	parsed, err := url.Parse(googleAppengineURL + "/" + name)
	if err != nil {
		log.Fatal(err)
	}
	return parsed.String()
}
