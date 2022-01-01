package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// googleSuggestion is one search query returned by the Google Suggest API.
type googleSuggestion struct {
	Data string `xml:"data,attr"`
}

// googleResults is an entire response returned by the Google Suggest API.
type googleResults struct {
	Suggestions []googleSuggestion `xml:"CompleteSuggestion>suggestion"`
}

// asStrings returns the suggestions in a GoogleResults object as plain strings for ease of use.
func (g *googleResults) asStrings() []string {
	var strings []string
	for _, s := range g.Suggestions {
		strings = append(strings, s.Data)
	}
	return strings
}

// nextOutputName counts the number of files that are in outputDir, and
// returns the file name for the next output file.
func nextOutputName(outputDir string) string {
	fis, err := ioutil.ReadDir(outputDir)
	if err != nil {
		log.Fatalln("could not read outputDir:", outputDir, "error:", err)
	}

	return path.Join(outputDir, fmt.Sprintf("results-%d.json", len(fis)))
}

// remainingQueries returns the list of remaining queries to crawl.
func remainingQueries(input, outputDir string) []*queryPackage {
	queries := make(map[string]*queryPackage)
	for _, pq := range constructQueries(loadPackageModules(input)) {
		queries[pq.query] = pq
	}

	var fetched int
	err := filepath.Walk(outputDir, func(path string, fi os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(in)
		for {
			var sugg curation.Suggestions
			err := decoder.Decode(&sugg)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			if _, exists := queries[sugg.Ident]; exists {
				delete(queries, sugg.Ident)
				fetched++
			}
		}

		return nil
	})

	if err != nil && err != io.EOF {
		log.Fatalln("error walking outputDir:", err)
	}

	log.Println("found", len(queries), "queries")
	log.Println(fetched, "already fetched, getting the rest...")

	var queryList []*queryPackage
	for _, pq := range queries {
		queryList = append(queryList, pq)
	}

	return queryList
}

// escape constructs a url-encoded query string to query the suggestions APIs.
func escape(name string) string {
	return url.QueryEscape(strings.Replace(name, ".", " ", -1))
}

// constructQueries reads in a list of PackageStates and
// generate queries for each package and the more popular modules .
func constructQueries(pstats []*pythoncode.PackageStats) []*queryPackage {
	var idents []*queryPackage
	for _, p := range pstats {
		// generate queries for the package
		idents = append(idents, &queryPackage{
			query:       join(prefix, p.Package),
			packageName: p.Package,
		})
		for i := 0; i < 26; i++ {
			idents = append(idents, &queryPackage{
				query:       join(prefix, p.Package, string(rune(offset+uint16(i)))),
				packageName: p.Package,
			})
		}

		// generate queries for the methods
		sort.Sort(sort.Reverse(pythoncode.MethodsByCount(p.Methods)))

		var total int
		for _, m := range p.Methods {
			total += m.Count
		}

		var accumulate int
		for _, m := range p.Methods {
			idents = append(idents, &queryPackage{
				query:       join(prefix, m.Ident),
				packageName: p.Package,
			})
			if float64(accumulate) < float64(total)*expansionCutoff {
				for i := 0; i < 26; i++ {
					idents = append(idents, &queryPackage{
						query:       join(prefix, m.Ident, string(rune(offset+uint16(i)))),
						packageName: p.Package,
					})
				}
			}
			accumulate += m.Count
		}
	}
	return idents
}

func join(items ...string) string {
	return strings.Join(items, " ")
}

// loadPackageModules loads an array of package stats from a file
func loadPackageModules(input string) []*pythoncode.PackageStats {
	r, err := fileutil.NewReader(input)
	if err != nil {
		log.Fatalln("error opening file:", err)
	}
	defer r.Close()

	var stats []*pythoncode.PackageStats
	decoder := json.NewDecoder(r)
	err = decoder.Decode(&stats)
	if err != nil {
		log.Fatal(err)
	}
	return stats
}

// fetchGoogleSuggestions queries Google for query suggestions
func fetchGoogleSuggestions(pq *queryPackage) (*curation.Suggestions, error) {
	url := fmt.Sprintf(endpoint + escape(pq.query))

	client := http.Client{}
	res, err := client.Get(url)

	if err != nil {
		log.Println("Error in GET request: ", err)
		return nil, err
	}

	results, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	switch {
	case err != nil:
		return nil, fmt.Errorf("error reading body of GET response: %v", err)
	case res.StatusCode > 400:
		return nil, fmt.Errorf("got status code: %d", res.StatusCode)
	default:
		return parseGoogle(pq, lang, source, results)
	}
}

func parseGoogle(pq *queryPackage, lang, source string, data []byte) (*curation.Suggestions, error) {
	// remove any invalid characters that are not UTF encoded
	data = curation.ValidUTF(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("error parsing google data, length 0")
	}

	var r googleResults
	err := xml.Unmarshal(data, &r)
	if err != nil {
		return nil, fmt.Errorf("error while parsing google xml: %v", err)
	}

	return &curation.Suggestions{
		Ident:       pq.query,
		Package:     pq.packageName,
		Language:    lang,
		Source:      source,
		Suggestions: r.asStrings(),
	}, nil
}
