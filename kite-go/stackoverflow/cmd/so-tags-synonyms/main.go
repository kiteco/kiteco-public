package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"golang.org/x/net/html"
)

const (
	rootURL = "http://stackoverflow.com/"
	url     = "http://stackoverflow.com/tags/synonyms"
)

var (
	pageRegex = regexp.MustCompile(`/tags/synonyms\?page=([0-9]+)&tab=newest&filter=all`)
	synRegex  = regexp.MustCompile(`synonym-([0-9])+`)
	pageTemp  = `/tags/synonyms?page=%d&tab=newest&filter=all`
)

func main() {
	var (
		outPath string
	)
	flag.StringVar(&outPath, "out", "", "path to write fetched synonyms")
	flag.Parse()
	if outPath == "" {
		flag.Usage()
		log.Fatal("out parameter REQUIRED")
	}

	// fetch the first page
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	tokenizer := html.NewTokenizer(resp.Body)
	// get all links that we need to crawl
	var maxPageNum int
	for {
		tt := tokenizer.Next()
		if tt == html.StartTagToken {
			t := tokenizer.Token()
			isLink := t.Data == "a"
			if !isLink {
				continue
			}
			for _, att := range t.Attr {
				if att.Key == "href" {
					if m := pageRegex.FindStringSubmatch(att.Val); len(m) > 0 {
						pagenum, err := strconv.Atoi(m[1])
						if err != nil {
							log.Println(err)
						}

						if pagenum > maxPageNum {
							maxPageNum = pagenum
						}
					}
					break
				}
			}
		}
		if tt == html.ErrorToken {
			break
		}
	}

	synonyms := make(map[string][]string)
	for i := 1; i <= maxPageNum; i++ {
		link := rootURL + fmt.Sprintf(pageTemp, i)
		fmt.Println(link)
		getSynonyms(link, synonyms)
	}

	out, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	encoder := json.NewEncoder(out)
	err = encoder.Encode(synonyms)
}

func getSynonyms(link string, synonyms map[string][]string) {
	// fetch the page
	resp, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken {
			t := tokenizer.Token()
			if t.Data == "tr" {
				for _, att := range t.Attr {
					if att.Key == "class" && synRegex.MatchString(att.Val) {
						// the next two tds are the synonyms (except when there are images as part of the tag)
						var tagsSeen int
						var tags []string
						for tagsSeen < 2 {
							tt := tokenizer.Next()
							if tt == html.StartTagToken {
								t := tokenizer.Token()
								if t.Data == "a" {
									tagsSeen++
									for tt != html.TextToken {
										tt = tokenizer.Next()
										if tt == html.ErrorToken {
											log.Fatal("malformed synonym table row")
										}
									}
									t = tokenizer.Token()
									tags = append(tags, t.Data)

								}
							}
						}
						synonyms[tags[0]] = append(synonyms[tags[0]], tags[1])
						synonyms[tags[1]] = append(synonyms[tags[1]], tags[0])
					}
				}
			}

		}
	}
}
