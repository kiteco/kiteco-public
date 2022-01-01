package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/text"
)

// This binary takes the output of so-crawler and filters out
// queries that don't have any so results or the total number of votes
// on the top 10 returned posts is below a threshold. The output
// is a map that maps a package name to an array of *curation.SuggestionScore.

const (
	defaultMinPostNum  = 10
	defaultMinVote     = 10
	defaultMinView     = 5000
	defaultMinPageVote = 10
	defaultMinPageView = 2500

	maxPostNum = 2000
	soURLRoot  = "http://stackoverflow.com/questions/"
)

// searchResults is the same as stackoverflow.SearchResults except that
// it also contains package information about the query.
type searchResults struct {
	Query   string
	Source  string
	Package string
	Results []stackoverflow.SearchResult
}

func main() {
	var (
		inputDir   string
		outputFile string
		minPostNum int
		source     string
	)
	flag.StringVar(&inputDir, "inputDir", "", "dir that contains the results of so-crawler")
	flag.StringVar(&outputFile, "outputFile", "", "output json file that contains queries for each observed package in inputDir (.json)")
	flag.StringVar(&source, "source", "so", "source of the query [so|google|all] (used for filtering)")
	flag.IntVar(&minPostNum, "minPost", defaultMinPostNum, "minimal number of posts to filter out a query")
	flag.Parse()

	if inputDir == "" || outputFile == "" {
		flag.Usage()
		log.Fatal("must specify -inputDir, -outputFile")
	}

	if source != "all" && source != "so" && source != "google" {
		log.Fatal("-source can only take [all, so, google]")
	}

	// for fetching SO pages from DB
	client, err := stackoverflow.NewClient(nil)
	if err != nil {
		log.Fatalf("error loading stackoverflow.Client: %s", err)
	}

	packageSuggestions := make(map[string][]*curation.SuggestionScore)
	seenSuggestions := make(map[string]map[string]struct{})

	count := 0
	err = filepath.Walk(inputDir, func(path string, fi os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(in)
		for {
			var sr searchResults
			err := decoder.Decode(&sr)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			if len(sr.Results) < minPostNum {
				continue
			}

			if _, exists := seenSuggestions[sr.Package]; !exists {
				seenSuggestions[sr.Package] = make(map[string]struct{})
			}

			var postIDs []int
			for _, r := range sr.Results {
				postIDs = append(postIDs, int(r.ID))
			}

			pages, err := client.PostsByID(postIDs)
			if err != nil {
				log.Println(err)
			}

			var view, vote int64
			for _, p := range pages {
				pageVote := p.GetQuestion().GetPost().GetScore()
				pageView := p.GetQuestion().GetPost().GetViewCount()

				for _, ans := range p.GetAnswers() {
					pageVote += ans.GetPost().GetScore()
				}

				vote += pageVote
				view += pageView

				title := p.GetQuestion().GetPost().GetTitle()
				if !validQuery(title) {
					continue
				}

				if _, found := seenSuggestions[sr.Package][title]; found {
					continue
				}
				seenSuggestions[sr.Package][title] = struct{}{}

				if source == "all" || source == "so" {
					packageSuggestions[sr.Package] = append(packageSuggestions[sr.Package], &curation.SuggestionScore{
						Query:     title,
						Tokens:    text.Lower(text.Stem(text.Tokenize(title))),
						Package:   sr.Package,
						ViewCount: int(pageView),
						VoteCount: int(pageVote),
						URL:       soURLRoot + strconv.FormatInt(p.GetQuestion().GetPost().GetId(), 10),
						Source:    "so",
					})
				}
			}

			if _, found := seenSuggestions[sr.Package][sr.Query]; found {
				continue
			}
			seenSuggestions[sr.Package][sr.Query] = struct{}{}

			if source == "all" || source == "google" {
				packageSuggestions[sr.Package] = append(packageSuggestions[sr.Package], &curation.SuggestionScore{
					Query:     sr.Query,
					Tokens:    text.Lower(text.Stem(text.Tokenize(sr.Query))),
					Package:   sr.Package,
					ViewCount: int(view),
					VoteCount: int(vote),
					Source:    "google",
				})
			}

			count++
			if count%500 == 0 {
				log.Printf("Processed %d google suggestion results\n", count)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	for p, suggestions := range packageSuggestions {
		sort.Sort(sort.Reverse(curation.ByScore(suggestions)))
		n := min(len(suggestions), maxPostNum)
		packageSuggestions[p] = suggestions[:n]
	}

	fout, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	encoder := json.NewEncoder(fout)
	err = encoder.Encode(packageSuggestions)
	if err != nil {
		log.Fatal(err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// validQuery removes queries that are about errors or explainations.
func validQuery(s string) bool {
	tokens := text.Lower(text.Tokenize(s))
	for _, tok := range tokens {
		switch tok {
		case "why", "error", "what", "explain":
			return false
		}
	}
	return true
}
