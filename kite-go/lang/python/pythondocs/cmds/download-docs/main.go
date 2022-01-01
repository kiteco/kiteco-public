package main

import (
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"errors"
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

// NB: Go does not support the cipher used by readthedocs.org. We're currently
// using a hack of restricting the version to get it to work here.

const (
	defaultDocsDatafilePath = pythondocs.DefaultDocsDatafilePath
	defaultRawDocsPath      = "/var/kite/data/docs/python/source/"

	readTheDocsURL = "http://readthedocs.org"
	rtdAPITemplate = "/api/v1/project/{0}/?format=json"

	logPrefix = "[download-docs] "
	logFlags  = log.LstdFlags | log.Lshortfile
)

func queryRTD(packageID int, origURL string) (*url.URL, error) {
	URL := readTheDocsURL + rtdAPITemplate
	URL = strings.Replace(URL, "{0}", strconv.Itoa(packageID), 1)
	time.Sleep(time.Second) // Throttling.
	log.Println("Querying RTD:", URL)
	response, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	respText, err := ioutil.ReadAll(response.Body)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(respText) == 0 {
		return nil, errors.New("Empty response")
	}
	var rtdResp pythondocs.RTDProjectInfo
	err = json.Unmarshal(respText, &rtdResp)
	if err != nil {
		return nil, err
	}
	newURL := rtdResp.DownloadFormatURLs.HTMLZip
	if newURL == "" {
		return nil, errors.New("Current RTD URL is missing")
	}
	if newURL == origURL {
		return nil, errors.New("Current RTD URL is the same as datafile's")
	}
	pURL, err := url.Parse(newURL)
	if err != nil {
		return nil, err
	}
	if pURL.Scheme == "" {
		pURL.Scheme = "http"
	}
	return pURL, nil
}

func main() {
	var (
		input  string
		output string
	)
	flag.StringVar(&input, "input", defaultDocsDatafilePath, "Directory in which to place documentation downloads")
	flag.StringVar(&output, "output", defaultRawDocsPath, "Directory in which to place documentation downloads")
	flag.Parse()

	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)

	err := os.MkdirAll(path.Dir(output), 0755)
	if err != nil {
		log.Fatal(err)
	}
	ddf, err := os.Open(input)
	if err != nil {
		log.Fatal(err)
	}
	defer ddf.Close()
	decomp, err := gzip.NewReader(ddf)
	if err != nil {
		log.Fatal(err)
	}
	defer decomp.Close()
	dec := json.NewDecoder(decomp)
	var downloadCount, entryCount int
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MaxVersion: tls.VersionTLS11,
		},
	}
	client := &http.Client{Transport: tr}
	startTime := time.Now()
	for {
		var pd pythondocs.PackageDescriptor
		err := dec.Decode(&pd)
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}
		// If there is an RTD entry, download the HTML zip.
		if pd.ReadTheDocsEntry != nil {
			log.Println("Found RTD entry for", pd.Name)
			func() {
				dURL := pd.ReadTheDocsEntry.DownloadFormatURLs.HTMLZip
				if dURL == "" {
					log.Println("No HTMLZip URL for", pd.Name)
					return
				}
				reqURL, err := url.Parse(dURL)
				if err != nil {
					log.Println(err)
					return
				}
				if reqURL.Scheme == "" {
					reqURL.Scheme = "http"
				}
				log.Println("GET:", reqURL)
				response, err := client.Get(reqURL.String())
				if err != nil {
					log.Println(err)
					return
				}
				if response.StatusCode != 200 {
					log.Println("Server responded with", response.Status)
					// Verify the URL dynamically with the RTD API.
					newURL, err := queryRTD(pd.ReadTheDocsEntry.ID, dURL)
					if err != nil {
						log.Println(err)
						return
					}
					response, err = client.Get(newURL.String())
					if err != nil {
						log.Println(err)
						return
					}
				}
				defer response.Body.Close()
				if response.ContentLength >= 0 {
					log.Println("Downloading", response.ContentLength, "bytes...")
				}
				filename := path.Base(response.Request.URL.Path)
				if err != nil {
					log.Println(err)
					return
				}
				if filename == "" {
					log.Println("No filename resolved for", reqURL.String())
				}
				buf, err := ioutil.ReadAll(response.Body)
				if err != nil {
					log.Println(err)
					return
				}
				if len(buf) == 0 {
					log.Println("Content is zero-length")
					return
				}
				outPath := path.Join(output, filename)
				err = ioutil.WriteFile(outPath, buf, 0755)
				if err != nil {
					log.Println(err)
					return
				}
				log.Println("Wrote to", outPath)
				downloadCount++
			}()
			// Sleep for a second so we don't DoS readthedocs.
			time.Sleep(time.Second)
		} else {
			log.Println("No RTD entry for", pd.Name)
		}

		// TODO(john/tarak): for the remainder, go through the common crawl starting
		// at the DocsURL for each entry and look for Sphinx docs.

		entryCount++
		log.Println("")
	}
	endTime := time.Now()
	log.Println("Downloaded", downloadCount, "of", entryCount)
	log.Println("Time taken:", endTime.Sub(startTime))
}
