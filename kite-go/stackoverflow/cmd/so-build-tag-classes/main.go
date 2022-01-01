package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
)

// extractAllTags iterates over the SO pages, stored in file with name pagesPath,
// and extracts all of the SO tags encountered and tracks the number of times a given tag is encountered.
func extractAllTags(pagesPath string) search.TagCount {
	f, err := os.Open(pagesPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	decoder := gob.NewDecoder(f)
	tags := make(search.TagCount)
	var page stackoverflow.StackOverflowPage
	for {
		err = decoder.Decode(&page)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		tokens := search.SplitTags(page.GetQuestion().GetPost().GetTags())
		for _, tok := range tokens {
			tags[tok]++
		}
	}
	return tags
}

// bfs returns all of the tags that are connected to tag.
// bfs = breadth first search
func bfs(tag string, allTags search.TagCount, allSynonyms map[string][]string) search.TagCount {
	toVisit := []string{tag}
	class := make(search.TagCount)
	for {
		if len(toVisit) == 0 {
			break
		}
		visit := toVisit[0]
		if _, exists := class[visit]; !exists {
			for _, syn := range allSynonyms[visit] {
				toVisit = append(toVisit, syn)
			}
			class[visit] = allTags[visit]
		}
		if len(toVisit) > 1 {
			toVisit = toVisit[1:]
		} else {
			toVisit = nil
		}
	}
	return class
}

// canonicalizeTags merges all tags that are synonyms to create tag classes, tags that do not
// have any synonyms each have their own tag class, each containing a single tag.
func canonicalizeTags(allTags search.TagCount, allSynonyms map[string][]string) search.TagClassData {
	tcd := search.TagClassData{
		TagClassIdx: make(map[string]int),
	}
	for tag := range allTags {
		if _, exists := tcd.TagClassIdx[tag]; exists {
			continue
		}
		class := bfs(tag, allTags, allSynonyms)
		for t := range class {
			tcd.TagClassIdx[t] = len(tcd.TagClasses)
		}
		tcd.TagClasses = append(tcd.TagClasses, class)
	}
	return tcd
}

func readSynonyms(synPath string) map[string][]string {
	f, err := os.Open(synPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	var syns map[string][]string
	err = decoder.Decode(&syns)
	if err != nil {
		log.Fatal(err)
	}
	return syns
}

// This binary takes as input a map of SO tag synonyms
// (map[string][]string) and an SO pages dump (in GOB format, one page per line)
// and writes a file containing the search.TagClassData assembled from the inputs, output is written in GOB format.
func main() {
	var (
		synPath   string
		pagesPath string
		outBase   string
	)
	flag.StringVar(&synPath, "syn", "", "path to so tag synonyms (REQUIRED)")
	flag.StringVar(&pagesPath, "pages", "", "path to so pages dump (REQUIRED)")
	flag.StringVar(&outBase, "out", "", "outpath to dump search.TagClassData to in GOB format (REQUIRED)")

	flag.Parse()
	if synPath == "" || outBase == "" || pagesPath == "" {
		flag.Usage()
		log.Fatal("syn, pages, and out parameters REQUIRED")
	}

	start := time.Now()
	tagSyns := readSynonyms(synPath)
	allTags := extractAllTags(pagesPath)
	tcd := canonicalizeTags(allTags, tagSyns)

	f, err := os.Create(outBase)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	encoder := gob.NewEncoder(f)
	err = encoder.Encode(tcd)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Done! Took ", time.Since(start))
}
