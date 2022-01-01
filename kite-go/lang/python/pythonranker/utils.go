package pythonranker

import (
	"math"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/text"
)

// filter removes tokens seen in matched from the input tokens.
func filter(tokens []string, matched map[string]struct{}) []string {
	var unmatched []string
	for _, t := range tokens {
		if _, seen := matched[t]; !seen {
			unmatched = append(unmatched, t)
		}
	}
	return unmatched
}

// logSumExp receives a slice of log scores: log(a), log(b), log(c)...
// and returns log(a + b + c....)
func logSumExp(logs []float64) float64 {
	var max float64
	for _, l := range logs {
		if l > max {
			max = l
		}
	}
	var sum float64
	for _, l := range logs {
		sum += math.Exp(l - max)
	}
	return max + math.Log(sum)
}

// sum sums all the entries in an input slice
func sum(slice []float64) float64 {
	var sum float64
	for _, value := range slice {
		sum += value
	}
	return sum
}

// wordCount keeps track the count for the word
type wordCount struct {
	word  string
	count int
}

type wordByCount []*wordCount

func (bc wordByCount) Len() int           { return len(bc) }
func (bc wordByCount) Swap(i, j int)      { bc[j], bc[i] = bc[i], bc[j] }
func (bc wordByCount) Less(i, j int) bool { return bc[i].count < bc[j].count }

// buildWordCount returns a slice containing the names of all methods
// found in any corpus and a slice of the words found in any corpus along with the
// number of documents the word appeared in, across any corpus.
func buildWordCount(corpora ...map[string][]string) ([]string, []*wordCount) {
	seenMethods := make(map[string]struct{})
	countMap := make(map[string]*wordCount)
	var wordCounts []*wordCount

	for _, corpus := range corpora {
		for n, docs := range corpus {
			seenMethods[n] = struct{}{}
			for _, doc := range docs {
				tokens := text.SearchTermProcessor.Apply(text.TokenizeNoCamel(doc))
				for _, t := range tokens {
					wc, exists := countMap[t]
					if !exists {
						wc = &wordCount{word: t}
						countMap[t] = wc
						wordCounts = append(wordCounts, wc)
					}
					wc.count++
				}
			}
		}
	}
	sort.Sort(wordByCount(wordCounts))

	var methods []string
	for m := range seenMethods {
		methods = append(methods, m)
	}

	return methods, wordCounts
}

// countWordToMethods counts the number of times we saw word w in the context of method m,
// these counts are stored in wordToMethods
func countWordToMethods(wordToMethods map[string][]float64, methodToID map[string]int, corpora ...map[string][]string) {
	for _, corpus := range corpora {
		for method, docs := range corpus {
			for _, doc := range docs {
				tokens := text.TFProcessor.Apply(text.TokenizeNoCamel(doc))
				for _, t := range tokens {
					if _, top := wordToMethods[t]; top {
						wordToMethods[t][methodToID[method]]++
					}
				}
			}
		}
	}
}

func mergeCorpus(corpuses ...map[string][]string) map[string][]string {
	mergedCorpus := make(map[string][]string)
	for _, corpus := range corpuses {
		for met, docs := range corpus {
			mergedCorpus[met] = append(mergedCorpus[met], docs...)
		}
	}
	return mergedCorpus
}
