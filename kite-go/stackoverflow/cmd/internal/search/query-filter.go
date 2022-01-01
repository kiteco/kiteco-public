package search

import (
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/languagemodel"
	"github.com/kiteco/kiteco/kite-golib/text"
)

var (
	// SupportedLanguages defines the languages that
	// can be detected by the LanguageDetector.
	SupportedLanguages = []string{
		"objective-c",
		"php",
		"ruby",
		"javascript",
		"python",
		"go",
		"java",
		"c++",
		"c",
		"bash",
	}
)

// LanguageDetector detects the language associated with a given query.
type LanguageDetector struct {
	scorer   *languagemodel.Scorer
	langSyns map[string]string
}

// NewLanguageDetector returns a new LanguageDetector.
func NewLanguageDetector(scorer *languagemodel.Scorer, tcd TagClassData) *LanguageDetector {
	langSyns := make(map[string]string)
	for _, lang := range SupportedLanguages {
		ci, exists := tcd.TagClassIdx[lang]
		if !exists {
			continue
		}
		for tag := range tcd.TagClasses[ci] {
			langSyns[tag] = lang
		}
	}
	return &LanguageDetector{
		scorer:   scorer,
		langSyns: langSyns,
	}
}

// Detect detects the language a query is referring to,
// returns the language detected for the query, and true if a language
// tag was found explicitly in the query and false otherwise.
func (ld LanguageDetector) Detect(query string) (string, bool) {
	tokens := strings.Split(query, " ")

	// 1) check for explicit language tags
	langs := make(map[string]struct{})
	for _, tok := range tokens {
		if lang, exists := ld.langSyns[tok]; exists {
			langs[lang] = struct{}{}
		}
	}
	if len(langs) > 0 {
		if len(langs) > 1 {
			log.Printf("query %s, detected multiple languages: %v \n", query, langs)
		}
		// heuristics
		// 1) check if python
		if _, exists := langs["python"]; exists {
			return "python", true
		}
		// 2) return langugae that has highest score for query
		// among detected languages
		posterior := ld.scorer.Posterior(tokens)
		var maxScore float64
		var maxLang string
		for lang := range langs {
			score := posterior[lang]
			if score > maxScore {
				maxScore = score
				maxLang = lang
			}
		}
		return maxLang, true
	}

	// 2) no explicit language tags so fall back to python
	return "python", false
}

// ResultFilter removes results that are deemed to be irrelevant to the given query.
type ResultFilter struct {
	tcd TagClassData
}

// NewResultFilter returns a new ResultFilter.
func NewResultFilter(tcd TagClassData) *ResultFilter {
	seps := []string{" ", ""}
	for tag, ci := range tcd.TagClassIdx {
		if !strings.Contains(tag, "-") {
			continue
		}
		parts := strings.Split(tag, "-")
		joinedTags := joinTokens(parts, seps)
		for _, joined := range joinedTags {
			tcd.TagClassIdx[joined] = ci
			tcd.TagClasses[ci][joined] = 1
		}
	}
	return &ResultFilter{
		tcd: tcd,
	}
}

// Filter removes SO pages that are not relevant to the given query.
func (rf ResultFilter) Filter(query, lang string, pages []*stackoverflow.StackOverflowPage) []*stackoverflow.StackOverflowPage {
	tokens := strings.Split(query, " ")
	tagClasses := make(map[int]struct{})
	// 1) check for explicit tags in the query
	// a) unigram tokens
	for _, tok := range tokens {
		ci, exists := rf.tcd.TagClassIdx[tok]
		if !exists {
			continue
		}
		tagClasses[ci] = struct{}{}
	}
	// b) bigram tokens
	if len(tokens) > 1 {
		seps := []string{"", " ", "-"}
		bigrams, _ := text.NGrams(2, tokens)
		for _, bg := range bigrams {
			joinedBGs := joinTokens(bg, seps)
			for _, joined := range joinedBGs {
				ci, exists := rf.tcd.TagClassIdx[joined]
				if !exists {
					continue
				}
				tagClasses[ci] = struct{}{}
			}
		}
	}

	// 2) check if we found any tag classes and try to add language tag class
	if len(tagClasses) == 0 {
		log.Printf("no tags detected for query %s \n", query)
	}

	langClassIdx, exists := rf.tcd.TagClassIdx[lang]
	if !exists {
		log.Printf("no tag class for language %s\n", lang)
	}
	tagClasses[langClassIdx] = struct{}{}

	if len(tagClasses) == 0 {
		return pages
	}

	// 3) remove pages that do not contain ANY of the tag classes
	// associated with the query
	var newPages []*stackoverflow.StackOverflowPage
	for _, page := range pages {
		tags := SplitTags(page.GetQuestion().GetPost().GetTags())
		for _, tag := range tags {
			ci, exists := rf.tcd.TagClassIdx[tag]
			if !exists {
				log.Printf("no class for tag %s\n", tag)
				continue
			}
			if _, exists := tagClasses[ci]; !exists {
				continue
			}
			newPages = append(newPages, page)
			break
		}
	}
	return newPages
}

func joinTokens(tokens, seps []string) []string {
	joined := make([]string, len(seps))
	for i, sep := range seps {
		joined[i] = strings.Join(tokens, sep)
	}
	return joined
}
