package main

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	defaultRTDDatafilePath    = pythondocs.DefaultRTDDatafilePath
	defaultDocsDatafilePath   = pythondocs.DefaultDocsDatafilePath
	defaultPackageCount       = 1000
	pyPIRankingURL            = "http://pypi-ranking.info"
	pyPIRankingURLTemplate    = pyPIRankingURL + "/alltime?page={0}"
	readTheDocsAPIURLTemplate = "http://readthedocs.org/api/v1/project/{0}/?format=json"

	logPrefix = "[crawl-python-docs] "
	logFlags  = log.LstdFlags | log.Lshortfile
)

var (
	rtdProjectNameMap = make(map[string]*pythondocs.RTDProjectInfo)
	rtdProjectHostMap = make(map[string]*pythondocs.RTDProjectInfo)

	errNoPackageInfo      = errors.New("No package information found")
	errNoPyPIPageURL      = errors.New("No PyPI page URL found")
	errNoPyPIModuleText   = errors.New("No text for module in PyPI page")
	errURLResponseNotHTML = errors.New("URL does not return HTML content")
)

// Counters for phase contributions.
var stats struct {
	RTDEntries int

	InitialRTDEntries                              int
	RepairedRTDEntries                             int
	RepairedRTDEntriesResolve                      int
	StructuredTableDocsURLs                        int
	StructuredTableDocsURLsResolve                 int
	StructuredTableHomepageURLs                    int
	DocsLinks                                      int
	DocsLinksResolve                               int
	HomepageLinksSearched                          int
	HomepageLinkURLs                               int
	OtherURLRepairedRTDEntries                     int
	OtherURLRepairedRTDEntriesResolve              int
	OtherURLRepairedRTDEntriesCanonicalURLs        int
	OtherURLRepairedRTDEntriesCanonicalURLsResolve int
	OtherURLDirectDocsURLs                         int
	OtherURLDirectDocsURLsResolve                  int
	OtherURLDocsLinkURLs                           int
	OtherURLDocsLinkURLsResolve                    int
	OtherURLGithubRepairedRTDEntries               int
	OtherURLGithubRepairedRTDEntriesResolve        int
	OtherURLGithubDocsLinkURLs                     int
	OtherURLGithubDocsLinkURLsResolve              int

	PotentiallyBad   int
	ResolvedDocsURLs int
	EntriesSum       int
}

func loadRTDDatafile(rtdDatafile string) error {
	rtddf, err := os.Open(rtdDatafile)
	if err != nil {
		return err
	}
	defer rtddf.Close()
	decomp, err := gzip.NewReader(rtddf)
	if err != nil {
		return err
	}
	defer decomp.Close()
	dec := json.NewDecoder(decomp)
	for {
		var rtdpi pythondocs.RTDProjectInfo
		err := dec.Decode(&rtdpi)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		rtdProjectNameMap[rtdpi.Name] = &rtdpi
		subURL, err := url.Parse(rtdpi.RTDSubdomain)
		if err != nil {
			log.Println(err)
			continue
		}
		rtdProjectHostMap[subURL.Host] = &rtdpi
	}
	log.Printf("Loaded RTD datafile %s (%d, %d entries)\n", rtdDatafile, len(rtdProjectNameMap), len(rtdProjectHostMap))
	return nil
}

func checkReadTheDocsEntryByName(name string) *pythondocs.RTDProjectInfo {
	return rtdProjectNameMap[name]
}

func checkReadTheDocsEntryByHost(host string) *pythondocs.RTDProjectInfo {
	return rtdProjectHostMap[host]
}

// Canonicalizes a given URL against a given referenced URL. Fragments/anchors
// are not retained (e.g., "index.html#footer" becomes "index.html").
func canonicalizeURL(URL string, refURL string) (*url.URL, error) {
	pURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	pRef, err := url.Parse(refURL)
	if err != nil {
		return nil, err
	}
	pCanon := pRef.ResolveReference(pURL)
	pCanon.Fragment = ""
	return pCanon, nil
}

func matchesDocsWords(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "documentation") ||
		strings.Contains(text, "docs") ||
		strings.Contains(text, "api")
}

func matchesHomepageWords(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "homepage") ||
		strings.Contains(text, "home page")
}

func matchesDocsSiteURL(text string) bool {
	text = strings.ToLower(text)
	return strings.HasSuffix(text, "readthedocs.org") ||
		strings.HasSuffix(text, "rtfd.org") ||
		strings.HasSuffix(text, "pythonhosted.org")
}

func isDocumentation(doc *goquery.Document) (bool, error) {
	h, err := doc.Html()
	if err != nil {
		return false, err
	}
	// Check for readthedocs script struct.
	if strings.Contains(h, "READTHEDOCS_DATA") {
		return true, nil
	}
	// Check for Sphinx footer.
	if strings.Contains(h, "Built with <a href=\"http://sphinx-doc.org/\">Sphinx</a>") {
		return true, nil
	}
	return false, nil
}

// Gets all the leading text before a node (in backwards order), up to the first
// encountered `a` node or the first encountered sentence end, and combines it
// with the text for the node itself. The text is not intended to be readable.
func combinedLeadingText(sel *goquery.Selection) string {
	ret := sel.Text()
	node := sel.Get(0)
	if node == nil {
		return ""
	}
	for prevSib := node.PrevSibling; prevSib != nil && prevSib.DataAtom != atom.A; prevSib = prevSib.PrevSibling {
		if prevSib.Type == html.TextNode {
			sei := strings.LastIndex(prevSib.Data, ". ")
			if sei == -1 {
				sei = strings.LastIndex(prevSib.Data, ".\n")
			}
			if sei != -1 {
				ret += " " + prevSib.Data[sei+2:]
				break
			}
			if strings.HasSuffix(prevSib.Data, ".") {
				break
			}
			ret += " " + prevSib.Data
		}
	}
	return strings.TrimSpace(ret)
}

func hostMatchesPackage(host string, packageName string) bool {
	host = strings.ToLower(host)
	packageName = strings.ToLower(packageName)
	hpi := strings.Index(host, ".readthedocs.org")
	if hpi == -1 {
		hpi = strings.Index(host, ".rtfd.org")
	}
	if hpi == -1 {
		return false
	}
	host = host[:hpi]
	if strings.Contains(host, packageName) || strings.Contains(packageName, host) {
		return true
	}
	return false
}

func findRTDLink(sel *goquery.Selection, URL string, packageName string) string {
	ret := ""
	sel.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		hrefURL, err := url.Parse(href)
		if err != nil {
			log.Println(err)
			return
		}
		if !strings.HasSuffix(hrefURL.Host, "readthedocs.org") {
			return
		}
		if !hostMatchesPackage(hrefURL.Host, packageName) {
			log.Println("Host does not match package name:", hrefURL.Host)
			return
		}
		rtde := checkReadTheDocsEntryByHost(hrefURL.Host)
		if rtde != nil {
			ret = hrefURL.String()
		}
	})
	return ret
}

// Finds documentation links within the given selection and canonicalizes them
// against the given URL. Returns the list of canonicalized URLs.
func findDocsLinks(sel *goquery.Selection, URL string) []string {
	var ret []string
	// Check for documentation text.
	sel.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}
		if strings.Contains(href, "python.org") ||
			strings.Contains(href, "confluence.atlassian.com") {
			return
		}
		// Check the link and leading text.
		if matchesDocsWords(combinedLeadingText(s)) {
			cURL, err := canonicalizeURL(href, URL)
			if err != nil {
				log.Println(err)
				return
			}
			if cURL.Fragment != "" {
				return
			}
			ret = append(ret, cURL.String())
			return
		}
	})
	return ret
}

func parseURL(URL string) (*goquery.Document, error) {
	var doc *goquery.Document
	if strings.HasPrefix(URL, "file://") {
		URL = strings.TrimPrefix(URL, "file://")
		file, err := os.Open(URL)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		doc, err = goquery.NewDocumentFromReader(file)
		if err != nil {
			return nil, err
		}
	} else {
		reqURL, err := url.Parse(URL)
		if err != nil {
			return nil, err
		}
		time.Sleep(time.Second) // Throttling.
		log.Println("GET:", URL)
		result, err := http.Get(reqURL.String())
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(result.Header.Get("Content-type"), "text/html") {
			return nil, errURLResponseNotHTML
		}
		doc, err = goquery.NewDocumentFromResponse(result)
		if err != nil {
			return nil, err
		}
	}
	// NewDocumentFromResponse closes result.Body for us.
	return doc, nil
}

// Gets the PyPI package page URL for a given pypi-ranking package page.
func getPyPIPackageURL(URL string) (string, error) {
	// Get the pypi-ranking page for this package.
	doc, err := parseURL(URL)
	if err != nil {
		return "", err
	}
	var pyPIURL string
	doc.Find("h2#item_title").Each(func(i int, s *goquery.Selection) {
		// Find the package's PyPI link in the page.
		a := s.Find("a").First()
		if a == nil {
			return
		}
		href, exists := a.Attr("href")
		if !exists || href == "" {
			return
		}
		pyPIURL = href
	})
	if pyPIURL == "" {
		return "", errNoPyPIPageURL
	}
	return pyPIURL, nil
}

// Given a URL, determines which parsing function to use and calls it.
// Intended to be used for URLs whose hosts we don't know ahead of time.
func parsePage(URL string, pd *pythondocs.PackageDescriptor) error {
	pURL, err := url.Parse(URL)
	if err != nil {
		return err
	}
	if strings.HasSuffix(pURL.Host, "pypi.python.org") {
		return parsePyPIPackagePage(URL, pd)
	} else if strings.HasSuffix(pURL.Host, "github.com") {
		return parseGitHubPage(URL, pd)
	}
	return parseOtherPage(URL, pd)
}

func parsePyPIPackagePage(URL string, pd *pythondocs.PackageDescriptor) error {
	// Follow the link.
	doc, err := parseURL(URL)
	if err != nil {
		return err
	}

	// Look for any useful links on this page.
	// Scan the whole page for any link with a docs site in it.
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		hrefURL, err := url.Parse(href)
		if err != nil {
			log.Println(err)
			return
		}
		if !matchesDocsSiteURL(hrefURL.Host) {
			return
		}
		if strings.Contains(hrefURL.Host, "readthedocs.org") || strings.Contains(hrefURL.Host, "rtfd.org") {
			if !hostMatchesPackage(hrefURL.Host, pd.Name) {
				log.Println("Host does not match package name:", hrefURL.Host)
				return
			}
			rtde := checkReadTheDocsEntryByHost(hrefURL.Host)
			if rtde != nil {
				pd.ReadTheDocsEntry = rtde
				if pd.DocsURL == "" {
					stats.RepairedRTDEntriesResolve++
				}
				pd.SetDocsURL(rtde.RTDSubdomain)
				stats.RepairedRTDEntries++
			}
		} else {
			if !strings.Contains(strings.ToLower(href), strings.ToLower(pd.Name)) {
				log.Println("URL does not contain project name:", href)
				return
			}
			pd.TrySetDocsURL(href)
		}
	})

	// Check the structured footer table.
	doc.Find("table.list").Each(func(i int, s *goquery.Selection) {
		sn := s.NextAll()
		if sn == nil {
			return
		}
		sn.Find("li").Each(func(i int, s *goquery.Selection) {
			si := s.Find("strong").First()
			if si == nil {
				return
			}
			a := si.Next()
			if a == nil {
				return
			}
			href, exists := a.Attr("href")
			if !exists || href == "" {
				return
			}
			sit := si.Text()
			switch sit {
			case "Documentation:":
				if pd.TrySetDocsURL(href) {
					stats.StructuredTableDocsURLsResolve++
				}
				stats.StructuredTableDocsURLs++
			case "Home Page:":
				if pd.PyPIEntry.HomepageURL == "" {
					pd.PyPIEntry.HomepageURL = href
					stats.StructuredTableHomepageURLs++
				}
			}
		})
	})

	// Get any other links in the module text.
	mtext := doc.Find("div.section")
	if mtext == nil {
		return errNoPyPIModuleText
	}
	mtext.Children().EachWithBreak(func(i int, s *goquery.Selection) bool {
		if len(s.Nodes) == 0 {
			return true
		}
		switch s.Nodes[0].DataAtom {
		case atom.Div:
			class, exists := s.Attr("class")
			if exists &&
				(class == "download-button" || class == "footer" || class == "credits") {
				return true
			}
		case atom.Table:
			// Stop when we've encountered the footer table.
			class, exists := s.Attr("class")
			if exists && class == "list" {
				return false
			}
		}

		// Find any other documentation links.
		docsURLs := findDocsLinks(s, URL)
		if len(docsURLs) > 0 {
			stats.DocsLinks++
		}
		for _, docsURL := range docsURLs {
			if pd.TrySetDocsURL(docsURL) {
				stats.DocsLinksResolve++
			}
		}

		// Check for homepage text.
		s.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists || href == "" {
				return
			}
			// Check the link and leading text.
			if matchesHomepageWords(combinedLeadingText(s)) {
				if pd.PyPIEntry.HomepageURL == "" {
					pd.PyPIEntry.HomepageURL = href
					stats.HomepageLinksSearched++
				}
				return
			}

			// No matches, so just add this URL to the OtherURLs list.
			cURL, err := canonicalizeURL(href, URL)
			if err != nil {
				log.Println(err)
				return
			}
			pd.PyPIEntry.OtherURLs = append(pd.PyPIEntry.OtherURLs, cURL.String())
		})
		return true
	})
	return nil
}

// Generic parser for page of unknown source.
func parseOtherPage(URL string, pd *pythondocs.PackageDescriptor) error {
	us, err := url.Parse(URL)
	if err != nil {
		return nil
	}
	// Don't parse python.org simple index pages.
	if strings.HasSuffix(us.Host, "python.org") &&
		strings.HasSuffix(strings.TrimSuffix(us.Path, "/"), "simple") {
		return nil
	}

	// Follow the link.
	doc, err := parseURL(URL)
	if err != nil {
		return err
	}

	isDoc, err := isDocumentation(doc)
	if err != nil {
		log.Println(err)
	}
	if isDoc {
		if pd.TrySetDocsURL(URL) {
			stats.OtherURLDirectDocsURLsResolve++
		}
		stats.OtherURLDirectDocsURLs++
	}

	// Look for an RTD canonical URL.
	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		rel, exists := s.Attr("rel")
		if !exists || rel != "canonical" {
			return
		}
		href, exists := s.Attr("href")
		if !exists || !strings.Contains(href, "readthedocs.org") {
			return
		}
		hrefURL, err := url.Parse(href)
		if err != nil {
			log.Println(err)
			return
		}
		if !hostMatchesPackage(hrefURL.Host, pd.Name) {
			log.Println("Host does not match package name:", hrefURL.Host)
			return
		}
		rtde := checkReadTheDocsEntryByHost(hrefURL.Host)
		if rtde != nil {
			pd.ReadTheDocsEntry = rtde
			if pd.DocsURL == "" {
				stats.OtherURLRepairedRTDEntriesCanonicalURLsResolve++
			}
			pd.SetDocsURL(href)
			stats.OtherURLRepairedRTDEntries++
			stats.OtherURLRepairedRTDEntriesCanonicalURLs++
		}
	})

	// Look for any useful links on this page.
	// Scan the whole page for any link with "readthedocs.org" in it.
	rtdURL := findRTDLink(doc.Selection, URL, pd.Name)
	if rtdURL != "" {
		pURL, err := canonicalizeURL(rtdURL, "")
		if err != nil {
			return err
		}
		rtde := checkReadTheDocsEntryByHost(pURL.Host)
		if rtde != nil {
			pd.ReadTheDocsEntry = rtde
			if pd.DocsURL == "" {
				stats.OtherURLRepairedRTDEntriesResolve++
			}
			pd.SetDocsURL(pURL.String())
			stats.OtherURLRepairedRTDEntries++
		}
	}
	// Find any other documentation links.
	docsURLs := findDocsLinks(doc.Selection, URL)
	if len(docsURLs) > 0 {
		stats.OtherURLDocsLinkURLs++
	}
	for _, docsURL := range docsURLs {
		if pd.TrySetDocsURL(docsURL) {
			stats.OtherURLDocsLinkURLsResolve++
		}
	}
	return nil
}

func parseGitHubPage(URL string, pd *pythondocs.PackageDescriptor) error {
	// Follow the link.
	doc, err := parseURL(URL)
	if err != nil {
		return err
	}
	doc.Find("div#readme").Each(func(i int, s *goquery.Selection) {
		// Look for any useful links on this page.
		// Scan the whole page for any link with "readthedocs.org" in it.
		rtdURL := findRTDLink(s, URL, pd.Name)
		if rtdURL != "" {
			pURL, err := canonicalizeURL(rtdURL, "")
			if err != nil {
				return
			}
			rtde := checkReadTheDocsEntryByHost(pURL.Host)
			if rtde != nil {
				pd.ReadTheDocsEntry = rtde
				stats.OtherURLGithubRepairedRTDEntries++
			}
			if pd.DocsURL == "" {
				stats.OtherURLGithubRepairedRTDEntriesResolve++
			}
			pd.SetDocsURL(pURL.String())
		}
		// Find any other documentation links.
		docsURLs := findDocsLinks(s, URL)
		if len(docsURLs) > 0 {
			stats.OtherURLGithubDocsLinkURLs++
		}
		for _, docsURL := range docsURLs {
			if pd.TrySetDocsURL(docsURL) {
				stats.OtherURLGithubDocsLinkURLsResolve++
			}
		}
	})
	return nil
}

func printStats() {
	out := fmt.Sprintf("%+v", stats)
	out = strings.Replace(out, " ", "\n", -1)
	out = strings.Replace(out, "{", "\n", -1)
	out = strings.Replace(out, "}", "", -1)
	out = strings.Replace(out, ":", ": ", -1)
	log.Println(out)
}

func main() {
	var (
		rtdDatafile string
		output      string
		numPackages int
	)
	flag.StringVar(&rtdDatafile, "rtdDatafile", defaultRTDDatafilePath, "Filename for readthedocs datafile")
	flag.StringVar(&output, "output", defaultDocsDatafilePath, "Filename for documentation datafile")
	flag.IntVar(&numPackages, "numPackages", defaultPackageCount, "The number of packages for which to obtain documentation (increments of 50)")
	flag.Parse()

	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)

	// Load in the RTD datafile.
	err := loadRTDDatafile(rtdDatafile)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare the output file.
	err = os.MkdirAll(path.Dir(output), 0755)
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

	startTime := time.Now()
	// Iterate until we hit N packages (each pypi-ranking page has 50 of these).
	for i, page := 0, 1; i < numPackages; i, page = i+50, page+1 {
		// Grab a page from pypi-ranking.
		queryURL := strings.Replace(pyPIRankingURLTemplate, "{0}", strconv.Itoa(page), 1)
		doc, err := parseURL(queryURL)
		if err != nil {
			log.Fatal(err)
		}

		// Go through each package found in this page.
		doc.Find("table").Each(func(i int, s *goquery.Selection) {
			id, exists := s.Attr("id")
			if !exists || id != "main_list" {
				return
			}
			s.Find("tr").Each(func(i int, s *goquery.Selection) {
				pd := pythondocs.NewPackageDescriptor("")
				var pprPageURL string
				s.Find("td").Each(func(i int, s *goquery.Selection) {
					class, exists := s.Attr("class")
					if !exists {
						return
					}
					switch class {
					case "rank":
						pd.PyPIEntry.Rank = s.Text()
					case "description":
						modulePageURL := s.Find("a").First()
						if modulePageURL == nil {
							return
						}
						href, exists := modulePageURL.Attr("href")
						if !exists || href == "" {
							return
						}
						pprPageURL = href

						listTitle := s.Find("span.list_title").First()
						if listTitle == nil {
							return
						}
						pd.Name = listTitle.Text()
						pd.PyPIEntry.Name = pd.Name

						listSummary := s.Find("p.list_summary_ellipsis").First()
						if listSummary == nil {
							return
						}
						pd.PyPIEntry.Summary = listSummary.Text()
					case "count":
						cn := s.Find("span").First()
						if cn == nil {
							return
						}
						count, err := strconv.Atoi(strings.Replace(cn.Text(), ",", "", -1))
						if err != nil {
							log.Println(err)
							return
						}
						pd.PyPIEntry.DownloadCount = count
					default:
						return
					}
				})
				if pd.Name == "" {
					return
				}
				log.Printf("[%d] Package: %s\n", i+1, pd.Name)

				rtde := checkReadTheDocsEntryByName(pd.Name)
				if rtde != nil {
					pd.ReadTheDocsEntry = rtde
					pd.SetDocsURL(rtde.RTDSubdomain)
					stats.InitialRTDEntries++
				}

				// Parse the pypi-ranking page for the PyPI page link.
				if pprPageURL != "" {
					ppru := pyPIRankingURL + pprPageURL
					pyPIURL, err := getPyPIPackageURL(ppru)
					if err != nil {
						log.Printf("Error parsing pypi-ranking page (%s): %s\n", ppru, err)
					}
					err = parsePyPIPackagePage(pyPIURL, pd)
					if err != nil {
						log.Printf("Error parsing PyPI page (%s): %s\n", pyPIURL, err)
					}
				}

				// Explore the homepage URL if we have it.
				hp := pd.PyPIEntry.HomepageURL
				if hp != "" {
					err = parsePage(hp, pd)
					if err != nil {
						log.Printf("Error parsing homepage (%s): %s\n", hp, err)
					}
				}

				// Explore any other URLs if we have them.
				for _, URL := range pd.PyPIEntry.OtherURLs {
					err = parsePage(URL, pd)
					if err != nil {
						log.Printf("Error parsing other page (%s): %s\n", URL, err)
					}
				}

				if pd.ReadTheDocsEntry != nil {
					stats.RTDEntries++
				}
				if pd.PyPIEntry.HomepageURL != "" {
					stats.HomepageLinkURLs++
				}
				if pd.DocsURL != "" {
					stats.ResolvedDocsURLs++
					log.Println("DocsURL:", pd.DocsURL)
				}
				stats.EntriesSum++
				pd.PyPIEntry.DocsURLs = removeDuplicates(pd.PyPIEntry.DocsURLs)
				pd.PyPIEntry.OtherURLs = removeDuplicates(pd.PyPIEntry.OtherURLs)
				enc.Encode(pd)
				log.Println("")
			})
		})
	}
	endTime := time.Now()
	log.Println("Time taken:", endTime.Sub(startTime))
	printStats()
}

func removeDuplicates(words []string) []string {
	var result []string
	seen := map[string]bool{}
	for _, val := range words {
		if !seen[val] {
			result = append(result, val)
			seen[val] = true
		}
	}
	return result
}
