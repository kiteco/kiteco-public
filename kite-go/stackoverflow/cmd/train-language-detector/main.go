package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
	"github.com/kiteco/kiteco/kite-golib/languagemodel"
	"github.com/kiteco/kiteco/kite-golib/text"
)

func langSynonyms(tagClassesPath string) map[string]string {
	f, err := os.Open(tagClassesPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	decoder := gob.NewDecoder(f)
	var tcd search.TagClassData
	err = decoder.Decode(&tcd)
	if err != nil {
		log.Fatal(err)
	}
	langSyns := make(map[string]string)
	for _, lang := range search.SupportedLanguages {
		ci, found := tcd.TagClassIdx[lang]
		if !found {
			log.Fatal("unable to find class for lang " + lang)
		}
		tagClass := tcd.TagClasses[ci]
		for tag := range tagClass {
			langSyns[tag] = lang
		}
		langSyns[lang] = lang
	}
	return langSyns
}

func main() {
	var (
		pagesPath      string
		tagClassesPath string
		outPath        string
	)
	flag.StringVar(&pagesPath, "pages", "", "path to GOB pages dump (REQUIRED)")
	flag.StringVar(&tagClassesPath, "tags", "", "path to tag class data in GOB format (REQUIRED)")
	flag.StringVar(&outPath, "out", "", "out path to write language scorer to (REQUIRED)")
	flag.Parse()
	if pagesPath == "" || tagClassesPath == "" || outPath == "" {
		flag.Usage()
		log.Fatal("pages, tags, out REQUIRED")
	}
	start := time.Now()

	langSyns := langSynonyms(tagClassesPath)

	f, err := os.Open(pagesPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	decoder := gob.NewDecoder(f)

	tokenizer := text.NewHTMLTokenizer()

	var docs []string
	var classes [][]string
	for {
		var page stackoverflow.StackOverflowPage
		err = decoder.Decode(&page)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		tags := text.Uniquify(search.SplitTags(page.GetQuestion().GetPost().GetTags()))
		var class []string
		for _, tag := range tags {
			if lang, exists := langSyns[tag]; exists {
				class = append(class, lang)
			}
		}
		if len(class) == 0 {
			continue
		}
		toks := tokenizer.Tokenize(page.GetQuestion().GetPost().GetBody())
		for _, ans := range page.GetAnswers() {
			toks = append(toks, tokenizer.Tokenize(ans.GetPost().GetBody())...)
		}
		doc := strings.Join(append(tags, toks...), " ")
		doc += " " + page.GetQuestion().GetPost().GetTitle()
		docs = append(docs, doc)
		classes = append(classes, class)
	}

	scorer, err := languagemodel.TrainScorer(docs, classes, tokenizer.Tokenize)
	if err != nil {
		log.Fatal(err)
	}

	fout, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()
	encoder := gob.NewEncoder(fout)
	err = encoder.Encode(scorer)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Done! took %v \n", time.Since(start))
}
