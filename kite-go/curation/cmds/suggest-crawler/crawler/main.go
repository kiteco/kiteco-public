package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/github"
)

const (
	perOutputFile   = 50
	numTriesPerName = 5
	sleepTime       = 20 // seconds
)

var (
	alphabet           = " abcdefghijklmnopqrstuvwxyz"
	sourcesToURLPrefix = map[string]string{
		"google": "http://suggestqueries.google.com/complete/search?output=toolbar&hl=en&q=",
		"bing":   "http://api.bing.com/osjson.aspx?query=",
	}
	client = &http.Client{
		Timeout: time.Duration(5) * time.Second,
	}
	skippedQueries []string
)

// --

func search(query string, sourceURL string) ([]byte, error) {
	for try := 0; try < numTriesPerName; try++ {
		res, err := client.Get(strings.ToLower(sourceURL + query))
		if err != nil {
			log.Println("Error in GET request: ", err)
		} else {
			results, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			switch {
			case err != nil:
				log.Println("Error reading body of GET response:", err)
			case res.StatusCode > 400:
				log.Println("Got status code:", res.StatusCode)
			default:
				return results, nil
			}
		}
		log.Printf("Try %d of %d in %d seconds...\n", try, numTriesPerName, sleepTime)
		time.Sleep(time.Duration(sleepTime) * time.Second) // sleep for 20 seconds
	}
	return nil, fmt.Errorf("all %d tries failed for query %s", numTriesPerName, query)
}

func combineSuggestions(name, lang, source string, suggestions []*curation.Suggestions) (*curation.Suggestions, error) {
	var merged []string
	seen := make(map[string]struct{})
	for _, s := range suggestions {
		for _, str := range s.Suggestions {
			if _, exists := seen[str]; !exists {
				merged = append(merged, str)
				seen[str] = struct{}{}
			}
		}
	}

	return &curation.Suggestions{
		Ident:       name,
		Language:    lang,
		Source:      source,
		Suggestions: merged,
	}, nil
}

func crawl(name string, source string) (*curation.Suggestions, error) {
	log.Println("Searching " + source + " for " + name)

	var suggestions []*curation.Suggestions
	for _, ch := range alphabet {
		time.Sleep(time.Duration(rand.Int31n(400)) * time.Millisecond) // sleep for b/w 0 and 2 secs randomly

		var query string
		if string(ch) != " " { // query will be '<name> <ch>', eg. 'os.join a', 'os.join b'
			query = constructQuery(strings.Join([]string{name, string(ch)}, " "))
		} else { // query will be '<name>', eg. 'os.join'
			query = constructQuery(name)
		}
		if len(query) > 100 { // google API returns bad request error
			log.Printf("Skipping query (more than 100 chars): %s\n", query)
			skippedQueries = append(skippedQueries, query)
			continue
		}
		data, err := search(query, sourcesToURLPrefix[source])
		if err != nil {
			return nil, err
		}
		s, err := parseSuggestions(name, "python", source, data)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, s)
	}

	log.Println("Done with ", name)

	return combineSuggestions(name, "python", source, suggestions)
}

func loadPackagesNamespace(input string) []string {
	stats := github.LoadPackageStats(input)

	var namespace []string
	for _, pkg := range stats {
		namespace = append(namespace, pkg.Name)
		for _, subPkg := range pkg.Submodules {
			namespace = append(namespace, subPkg.Name)
			for _, method := range subPkg.Methods {
				namespace = append(namespace, method.Name)
			}
		}
	}

	var clean []string
	for _, name := range namespace {
		if name != "" {
			clean = append(clean, name)
		}
	}

	return clean
}

func selectSpecifiedPackages(input string, packages []string) []github.Package {
	var selected []github.Package
	for _, pkg := range github.LoadPackageStats(input) {
		for _, chosen := range packages {
			if pkg.Name == strings.TrimSpace(chosen) {
				log.Printf("Found package %s. Wrote out.\n", pkg.Name)
				selected = append(selected, *pkg)
			}
		}
	}
	return selected
}

func getRemainingPackages(input string, crawled []string) []github.Package {
	allNames := make(map[string]struct{})
	for _, name := range crawled {
		allNames[name] = struct{}{}
	}

	var selected []github.Package
	for _, pkg := range github.LoadPackageStats(input) {
		var dup github.Package
		if _, ok := allNames[pkg.Name]; !ok {
			dup.Name = pkg.Name
		}
		for _, subPkg := range pkg.Submodules {
			newSubPkg := &github.Submodule{}
			if _, ok := allNames[subPkg.Name]; !ok {
				newSubPkg.Name = subPkg.Name
			}
			for _, method := range subPkg.Methods {
				newMethod := &github.Method{}
				if _, ok := allNames[method.Name]; !ok {
					newMethod.Name = method.Name
				}
				newSubPkg.Methods = append(newSubPkg.Methods, newMethod)
			}
			dup.Submodules = append(dup.Submodules, newSubPkg)
		}
		selected = append(selected, dup)
	}
	return selected
}

func getCrawledNames(outputdirs []string) ([]string, error) {
	crawled := make(map[string]struct{})
	// go through each output dir and collect names in there
	for _, dir := range outputdirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			decomp, err := gzip.NewReader(in)
			if err != nil {
				return err
			}

			decoder := json.NewDecoder(decomp)
			for {
				var suggestions curation.Suggestions
				err := decoder.Decode(&suggestions)
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				crawled[suggestions.Ident] = struct{}{}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	var names []string
	for ident := range crawled {
		names = append(names, ident)
	}
	return names, nil
}

func getRemainingNames(namespace, crawled []string) []string {
	allNames := make(map[string]bool)
	for _, ident := range namespace {
		allNames[ident] = false
	}
	for _, ident := range crawled {
		allNames[ident] = true
	}

	var remaining []string
	for ident, crawled := range allNames {
		if !crawled {
			remaining = append(remaining, ident)
		}
	}
	return remaining
}

func newCompressedSink(path string) (*json.Encoder, *gzip.Writer, *os.File) {
	log.Printf("Creating new output file %s\n", path)
	out, err := os.Create(path)
	if err != nil {
		log.Printf("Error creating output file %s\n", path)
		return nil, nil, nil
	}
	comp := gzip.NewWriter(out)
	enc := json.NewEncoder(comp)
	return enc, comp, out
}

func closeCompressedSink(gz *gzip.Writer, f *os.File) error {
	if gz != nil {
		if err := gz.Close(); err != nil {
			log.Println("Error flushing gz writer")
			return err
		}
	}
	if f != nil {
		if err := f.Close(); err != nil {
			log.Println("Error closing file")
			return err
		}
	}
	return nil
}

func filesInDir(dir string) int {
	if fi, err := ioutil.ReadDir(dir); err == nil {
		return len(fi)
	}
	return 0
}

// --

func main() {
	var action, input, output, packages, source string
	flag.StringVar(&action, "action", "", "select or crawl or remaining or crawled")
	flag.StringVar(&input, "input", "", "packages/subpackages/methods file (json) eg. packages.stats.json")
	flag.StringVar(&output, "output", "", "file (for select), dir (for crawl, must already exist), or comma separated dirs (for remaining or crawled)")
	flag.StringVar(&source, "source", "", "google or bing")
	flag.StringVar(&packages, "packages", "", "comma separated list of packages that you want to select (not used for crawl)")
	flag.Parse()

	if action == "select" { // selects the specified packages from the input and outputs to new stats file
		selected := selectSpecifiedPackages(input, strings.Split(packages, ","))

		f, err := os.Create(output)
		if err != nil {
			log.Fatalf("Can't create for writing: %s\n", output)
		}
		enc := json.NewEncoder(f)

		for _, suggest := range selected {
			err := enc.Encode(suggest)
			if err != nil {
				log.Printf("Error while encoding suggestion\n")
			}
		}

		f.Close()

		log.Printf("Wrote selected packages out to file %s\n", output)
	} else if action == "crawl" {
		log.Println("Loading packages namespace...")
		namespace := loadPackagesNamespace(input)
		crawled, err := getCrawledNames([]string{output})
		if err != nil {
			log.Fatalf("Error getting list of crawled names: %v\n", err)
		}
		remaining := getRemainingNames(namespace, crawled)
		sort.Strings(remaining) // ensure same order each time the crawler is resumed on same input
		numFilesInOutput := filesInDir(output)

		var outputFile string
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		go func() {
			<-signalChan
			log.Println("Received INTERRUPT.")
			if err := os.Remove(outputFile); err != nil {
				log.Fatalf("Error removing last file %s before exiting. Skipped %d queries: %v\n", outputFile, len(skippedQueries), skippedQueries)
			} else {
				log.Fatalf("Removed last file %s before exiting. Skipped %d queries: %v\n", outputFile, len(skippedQueries), skippedQueries)
			}
		}()

		var enc *json.Encoder
		var gz *gzip.Writer
		var f *os.File

		log.Println("Crawling suggestions for each name...")
		for idx, name := range remaining {
			if idx%perOutputFile == 0 { // flush and close old file and create new file, every perOutputFile names
				if err := closeCompressedSink(gz, f); err != nil {
					log.Fatalf("Error closing output sink %d. Skipped %d queries: %v\n", (idx/perOutputFile)-1, len(skippedQueries), skippedQueries)
				}
				outputFile = filepath.Join(output, fmt.Sprintf("output-%d.json.gz", numFilesInOutput+(idx/perOutputFile)+1))
				enc, gz, f = newCompressedSink(outputFile)
				if enc == nil && gz == nil && f == nil {
					log.Fatalf("Error creating new output file %d. Skipped %d queries: %v\n", numFilesInOutput+(idx/perOutputFile)+1, len(skippedQueries), skippedQueries)
				}
			}
			suggest, err := crawl(name, source)
			if err != nil { // server not reachable, or banned, or internet off, or error parsing response
				log.Println(err.Error())
				if err := os.Remove(outputFile); err != nil {
					log.Fatalf("Searching %s failed. Error removing last file %s before exiting. Skipped %d queries: %v\n", name, outputFile, len(skippedQueries), skippedQueries)
				}
				log.Fatalf("Searching %s failed. Deleted last file %s. Exiting. Skipped %d queries: %v\n", name, outputFile, len(skippedQueries), skippedQueries)
			}
			err = enc.Encode(suggest)
			if err != nil {
				log.Fatalf("Error while encoding suggestion for %s. Skipped %d queries: %v\n", name, len(skippedQueries), skippedQueries)
			}
		}

		if gz != nil {
			gz.Close()
		}
		if f != nil {
			f.Close()
		}
	} else if action == "remaining" {
		// goes through output folders to find what has already been crawled,
		// and creates a new stats file with only those packages in the input
		// that have not already been crawled
		crawled, err := getCrawledNames(strings.Split(output, ","))
		if err != nil {
			log.Fatalf("Error while getting crawled names: %v", err)
		}
		remaining := getRemainingPackages(input, crawled)

		split := strings.Split(input, "/")
		name := strings.Split(split[len(split)-1], ".")[0]
		out := fmt.Sprintf("%s-remaining.stats.json", name)
		f, err := os.Create(out)
		if err != nil {
			log.Fatalf("Can't create for writing: %s\n", out)
		}
		enc := json.NewEncoder(f)

		for _, suggest := range remaining {
			err := enc.Encode(suggest)
			if err != nil {
				log.Printf("Error while encoding suggestion\n")
			}
		}

		f.Close()

		log.Printf("Wrote remaining packages out to file %s\n", out)
	} else if action == "crawled" {
		crawledNames, err := getCrawledNames(strings.Split(output, ","))
		if err != nil {
			log.Fatalf("Error while getting crawled names: %v", err)
		}

		crawledPkgs := make(map[string]struct{})
		for _, name := range crawledNames {
			log.Println(name)
			pkg := strings.Split(name, ".")[0]
			crawledPkgs[pkg] = struct{}{}
		}

		for pkg := range crawledPkgs {
			log.Printf("%s,", pkg)
		}
	}
}
