package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
)

const (
	defaultDatafilePath      = pythondocs.DefaultRTDDatafilePath
	defaultPackageCount      = 1000000
	defaultRTDIterationLimit = 1000
	readTheDocsURL           = "http://readthedocs.org"
	rtdAPIInitialTemplate    = "/api/v1/project/?format=json&limit={0}&offset=0"

	logPrefix = "[crawl-readthedocs] "
	logFlags  = log.LstdFlags | log.Lshortfile
)

func main() {
	var (
		output       string
		numPackages  int
		rtdIterLimit int
	)
	flag.StringVar(&output, "output", defaultDatafilePath, "Filename for readthedocs datafile")
	flag.IntVar(&numPackages, "numPackages", defaultPackageCount, "The number of packages for which to obtain readthedocs information (increments of rtdIterLimit)")
	flag.IntVar(&rtdIterLimit, "rtdIterLimit", defaultRTDIterationLimit, "The maximum number of packages to obtain on each request from readthedocs")
	flag.Parse()

	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)

	err := os.MkdirAll(path.Dir(output), 0755)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	comp := gzip.NewWriter(f)
	defer comp.Close()
	enc := json.NewEncoder(comp)

	URL := readTheDocsURL + rtdAPIInitialTemplate
	URL = strings.Replace(URL, "{0}", strconv.Itoa(rtdIterLimit), 1)
	for i := 0; i < numPackages; i += rtdIterLimit {
		reqURL, err := url.Parse(URL)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("GET:", reqURL)
		result, err := http.Get(reqURL.String())
		if err != nil {
			log.Fatal(err)
		}
		respText, err := ioutil.ReadAll(result.Body)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		if len(respText) == 0 {
			log.Fatal("Empty response from", reqURL)
		}
		result.Body.Close()
		var rtdResp pythondocs.RTDProjectsAPIResponse
		err = json.Unmarshal(respText, &rtdResp)
		if err != nil {
			log.Fatal(err)
		}
		for _, project := range rtdResp.Objects {
			enc.Encode(project)
		}
		if i+rtdIterLimit >= rtdResp.Meta.TotalCount || rtdResp.Meta.Next == "" {
			break
		}
		URL = readTheDocsURL + rtdResp.Meta.Next

		// Sleep for a second so we don't DoS readthedocs.
		time.Sleep(time.Second)
	}
}
