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
	pyqt4DocsURL = "http://pyqt.sourceforge.net/Docs/PyQt4"
	logPrefix    = "[crawl-pyqt4]"
	logFlags     = log.LstdFlags | log.Lshortfile
	index        = "modules.html"
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
		log.Fatalf("Error using goquery to parse response: %v", err)
	}
	header := doc.Find("h1")
	if header.Length() != 1 {
		log.Fatalf("Expect exactly one <h1>, got %d", header.Length())
	}
	rows := header.NextAllFiltered("table").Find("tr")
	if rows == nil {
		log.Fatalf("Expect rows with modules")
	}
	modules := make(map[string]string)
	rows.Each(func(i int, s *goquery.Selection) {
		td := s.ChildrenFiltered("td").First()
		var location string
		if anchor := td.Find("a"); anchor != nil {
			location, _ = anchor.Attr("href")
		}
		if strings.HasPrefix(location, "#") {
			return
		}
		modules[td.Text()] = location
	})

	for module, loc := range modules {
		crawlModule(module, loc)
	}
}

func crawlModule(module, url string) {
	if url == "" {
		return
	}

	buf := fetchAndSave(url)

	// Extract classes in module from index
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(buf))
	if err != nil {
		log.Fatalf("Error using goquery to parse response: %v", err)
	}

	classes := make(map[string]string)
	doc.Find("ul").First().Find("li").Each(func(i int, s *goquery.Selection) {
		anchor := s.Find("a")
		if anchor == nil {
			return
		}
		loc, exists := anchor.Attr("href")
		if !exists {
			return
		}
		if strings.HasPrefix(loc, "#") {
			return
		}
		classes[anchor.Text()] = loc
	})

	for _, loc := range classes {
		if loc == "" {
			continue
		}
		fetchAndSave(loc)
	}
}

func fetchAndSave(name string) []byte {
	docURL := makeDocURL(name)
	result, err := http.Get(docURL)
	if err != nil {
		log.Fatalf("Error getting URL: %v", err)
	}
	defer result.Body.Close()

	buf, err := ioutil.ReadAll(result.Body)
	if err != nil && err != io.EOF {
		log.Fatalf("Error reading bytes from response body: %v", err)
	}

	saveFile(filepath.Join(outputDir, name), buf)

	return buf
}

func makeDocURL(name string) string {
	parsed, err := url.Parse(pyqt4DocsURL + "/" + name)
	if err != nil {
		log.Fatal(err)
	}
	return parsed.String()
}

func saveFile(name string, buf []byte) {
	f, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.Write(buf)
}
