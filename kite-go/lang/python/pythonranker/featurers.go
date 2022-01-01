package pythonranker

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"io"
	"log"
	"math"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker/internal/precompute"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/languagemodel"
	"github.com/kiteco/kiteco/kite-golib/text"
	"github.com/kiteco/kiteco/kite-golib/tfidf"
)

const (
	minTokenLenForFuzzyMatching = 3
	minLogProb                  = -30
	backgroundLMProb            = 0.4
	minWordCount                = 5
)

var (
	lmProcessor = text.NewProcessor(text.Lower, text.RemoveStopWords, text.Stem)
)

// PackageFeaturer converts a query to a feature vector.
// The features we use now are:
// 1) exact package name matching
// 2) exact function name matching
// 3) fuzzy package name matching
// 4) fuzzy function name matching
// 5) (1 | 3) & (2 | 4) -> wehther the query contains both package and method name
// 6) if any query token matches with other package names
// 7) lm model posterior probability p(pkg|query)
// 8) tfidf score from doc data
// 9) tfidf score from so data
// 10) log package prior distribution from github data
type PackageFeaturer struct {
	PackageList             map[string]struct{}
	PackageNameMatcher      map[string]*KeywordMatcher
	FuzzyPackageNameMatcher map[string]*FuzzyKeywordMatcher
	SelNameMatcher          map[string]*KeywordMatcher
	FuzzySelNameMatcher     map[string]*FuzzyKeywordMatcher
	PackagePrior            map[string]float64
	DocLM                   *languagemodel.Scorer
	DocTFIDF                *tfidf.Scorer
	SoLM                    *languagemodel.Scorer
	SoTFIDF                 *tfidf.Scorer
}

// NewPackageFeaturer returns a pointer to a new PackageFeaturer object.
// It loads all the data needed for constructing the featurizer.
// The data are all maps from a package name to text data collected for
// the method from various corpuses. For example, packageSos maps
// from a package name to all the titles on SO we found for that package.
// list is the list of packages that are covered by the package ranker.
func NewPackageFeaturer(list map[string]struct{}, packageSos, packageDocs,
	packageSels map[string][]string, packagePrior map[string]float64) (*PackageFeaturer, error) {

	packageNameMatcher := make(map[string]*KeywordMatcher)
	selNameMatcher := make(map[string]*KeywordMatcher)
	fuzzyPackageNameMatcher := make(map[string]*FuzzyKeywordMatcher)
	fuzzySelNameMatcher := make(map[string]*FuzzyKeywordMatcher)

	for p, sels := range packageSels {
		// keyword matchers for package
		packageNameMatcher[p] = newKeywordMatcher(text.Lower([]string{p}))
		fuzzyPackageNameMatcher[p] = newFuzzyKeywordMatcher([]string{p})

		// keyword matchers for selectors
		selNameMatcher[p] = newKeywordMatcher(text.Lower(sels))
		fuzzySelNameMatcher[p] = newFuzzyKeywordMatcher(sels)
	}

	docLM, err := languagemodel.TrainScorerFromMap(packageDocs, text.Tokenize)
	if err != nil {
		return nil, err
	}

	docTFIDF := tfidf.TrainScorer(packageDocs, false, text.Tokenize)

	soLM, err := languagemodel.TrainScorerFromMap(packageSos, text.Tokenize)
	if err != nil {
		return nil, err
	}
	soTFIDF := tfidf.TrainScorer(packageSos, true, text.Tokenize)

	return &PackageFeaturer{
		PackageList:             list,
		PackageNameMatcher:      packageNameMatcher,
		FuzzyPackageNameMatcher: fuzzyPackageNameMatcher,

		PackagePrior: packagePrior,

		SelNameMatcher:      selNameMatcher,
		FuzzySelNameMatcher: fuzzySelNameMatcher,
		DocLM:               docLM,
		DocTFIDF:            docTFIDF,
		SoLM:                soLM,
		SoTFIDF:             soTFIDF,
	}, nil
}

// NewPackageFeaturerFromJSON loads the featurer from a json file.
func NewPackageFeaturerFromJSON(r io.Reader) (*PackageFeaturer, error) {
	decoder := json.NewDecoder(r)
	var pf PackageFeaturer
	err := decoder.Decode(&pf)
	if err != nil {
		return nil, err
	}
	pf.DocLM.SetTextProcessors()
	return &pf, nil
}

// Labels returns the names of the features
func (p *PackageFeaturer) Labels() []string {
	return []string{
		"exactmatch_package",
		"exactmatch_selector",
		"fuzzymatch_package",
		"fuzzymatch_selector",
		"matched_package_selector",
		"unmatched_package",
		"lm",
		"doc_tfidf",
		"so_tfidf",
		"prior_github",
	}
}

// Find finds packages that contain the query tokens as selector names.
// It sorts the packages by popularity except that it always makes builtins
// the first choice (hack).
func (p *PackageFeaturer) Find(query string) []*ranking.DataPoint {
	tokens := strings.Split(query, " ")
	tokens = text.Lower(tokens)

	var dataPoints []*ranking.DataPoint
	for pkg := range p.PackageList {
		_, matched := p.SelNameMatcher[pkg].match(tokens)
		if len(matched) == len(tokens) {
			dataPoints = append(dataPoints, &ranking.DataPoint{
				Score: p.PackagePrior[pkg],
				Name:  pkg,
			})
			if pkg == "builtins" {
				dataPoints[len(dataPoints)-1].Score = 1
			}
		}
	}
	return dataPoints
}

// Features converts the query string into a set of feature vectors,
// where each feature vector corresonds to a package.
func (p *PackageFeaturer) Features(query string) []*ranking.DataPoint {
	queryTokens := text.Tokenize(query)

	cleanTokens := text.RemoveStopWords(text.Stem(text.Tokenize(query)))
	lmScores := p.DocLM.Posterior(cleanTokens)

	var dataPoints []*ranking.DataPoint
	for pkg := range p.PackageList {
		var matchedPackage bool
		var matchedMethod bool
		var feats []float64
		// match package names
		feat, matched := p.PackageNameMatcher[pkg].match(queryTokens)
		if feat > 1 {
			feat = 1
		}
		feats = append(feats, feat)
		if feat == 1 {
			matchedPackage = true
		}

		// remove words that were already matched and then match with function names
		unmatched := filter(queryTokens, matched)
		feat, matched = p.SelNameMatcher[pkg].match(unmatched)
		if feat > 1 {
			feat = 1
		}
		feats = append(feats, feat)
		if feat == 1 {
			matchedMethod = true
		}

		// fuzzy match package names
		unmatched = filter(unmatched, matched)
		feat, matched = p.FuzzyPackageNameMatcher[pkg].match(unmatched)
		if feat > 1 {
			feat = 1
		}
		feats = append(feats, feat)
		if feat == 1 {
			matchedPackage = true
		}

		// remove words that were already matched and then do fuzzy matching with function names
		unmatched = filter(unmatched, matched)
		feat, matched = p.FuzzySelNameMatcher[pkg].match(unmatched)
		if feat > 1 {
			feat = 1
		}
		feats = append(feats, feat)
		if feat == 1 {
			matchedMethod = true
		}

		// the query matched both package name and method name
		if matchedPackage && matchedMethod {
			feat = 1
		} else {
			feat = 0
		}
		feats = append(feats, feat)

		// check if any unmatched word is package name
		feat = 0
		for _, tok := range unmatched {
			if _, found := p.PackageList[tok]; found {
				feat = 1
			}
		}
		feats = append(feats, feat)

		// lm score
		feats = append(feats, lmScores[pkg])

		// doc tfidf score
		feats = append(feats, p.DocTFIDF.TFIDFScore(cleanTokens, pkg))

		// so tfidf score
		feats = append(feats, p.SoTFIDF.TFIDFScore(cleanTokens, pkg))

		// package prior distribution
		feats = append(feats, math.Log10(p.PackagePrior[pkg]))

		dataPoints = append(dataPoints, &ranking.DataPoint{
			Name:     pkg,
			Features: feats,
		})
	}
	return dataPoints
}

// KeywordMatcher contains a list of keywords so that it's easy
// to look up whether the query tokens contain any of the keywords.
type KeywordMatcher struct {
	Keywords map[string]struct{}
}

func newKeywordMatcher(words []string) *KeywordMatcher {
	keywords := make(map[string]struct{})
	for _, w := range words {
		keywords[w] = struct{}{}
	}
	return &KeywordMatcher{
		Keywords: keywords,
	}
}

// match returns returns 1.0 if any the input tokens match with the keywords.
// It also returns a list of matched tokens.
func (km *KeywordMatcher) match(tokens []string) (float64, map[string]struct{}) {
	seenTokens := make(map[string]struct{})
	for _, tok := range tokens {
		if _, seen := km.Keywords[tok]; seen {
			seenTokens[tok] = struct{}{}
		}
	}
	return float64(len(seenTokens)), seenTokens
}

// FuzzyKeywordMatcher does fuzzy keyword matching.
type FuzzyKeywordMatcher struct {
	Keywords map[string]struct{}
}

func newFuzzyKeywordMatcher(words []string) *FuzzyKeywordMatcher {
	keywords := make(map[string]struct{})
	for _, w := range words {
		terms := text.RemoveStopWords(text.Tokenize(w))
		for _, t := range terms {
			keywords[t] = struct{}{}
		}
	}
	return &FuzzyKeywordMatcher{
		Keywords: keywords,
	}
}

// match returns 1.0 if any of the input tokens matches with the keywords fussily.
func (km *FuzzyKeywordMatcher) match(tokens []string) (float64, map[string]struct{}) {
	seenTokens := make(map[string]struct{})
	for _, tok := range tokens {
		for k := range km.Keywords {
			if len(tok) >= minTokenLenForFuzzyMatching && (strings.HasSuffix(k, tok) || strings.HasPrefix(k, tok)) {
				seenTokens[tok] = struct{}{}
			}
			if len(tok) >= minTokenLenForFuzzyMatching && (strings.HasSuffix(tok, k) || strings.HasPrefix(tok, k)) {
				seenTokens[tok] = struct{}{}
			}
		}
	}
	return float64(len(seenTokens)), seenTokens
}

// --

// Featurer defines what a featurizer must satisfy.
type Featurer interface {
	Features(string, *precompute.QueryStats, *precompute.TargetStats) float64
	Label() string
}

// NewMethodFeaturerFromFile loads a pre-trained method featurer from file
func NewMethodFeaturerFromFile(r io.Reader) (map[string]*MethodFeaturer, error) {
	decomp, err := gzip.NewReader(r)
	if err != nil {
		log.Fatal(err)
	}

	decoder := gob.NewDecoder(decomp)
	featurers := make(map[string]*MethodFeaturer)
	err = decoder.Decode(&featurers)
	if err != nil {
		return nil, err
	}
	return featurers, nil
}

// MethodFeaturer generates features for a given method and a query. MethodFeaturer is
// trained on a per-package basis.
type MethodFeaturer struct {
	Featurers []Featurer
}

// NewMethodFeaturer takes in the training data which consists of maps that
// map a method name to its docs and return a pointer to a new
// MethodFeaturer object.
func NewMethodFeaturer(logPrior map[string]float64, chainedlogPrior map[string]float64, names, docCorpus,
	soCorpus, curationCorpus, keywords map[string][]string, pkg string) *MethodFeaturer {

	var featurers []Featurer

	mergedCorpus := mergeCorpus(soCorpus, curationCorpus, docCorpus)

	featurers = append(featurers, newMatchFeaturer(names, "exact_match"))
	featurers = append(featurers, newFuzzyMatchFeaturer(names, "fuzzy_match"))
	featurers = append(featurers, newTFIDFFeaturer(mergedCorpus, "tfidf_merged"))
	featurers = append(featurers, newLMFeaturer(mergedCorpus, "lm_merged"))
	featurers = append(featurers, newWordToMethodFeaturer(soCorpus, docCorpus, curationCorpus, "word_to_method"))
	featurers = append(featurers, newMatchFeaturer(keywords, "exact_keyword"))
	featurers = append(featurers, newDepthFeaturer("method_depth"))

	return &MethodFeaturer{
		Featurers: featurers,
	}
}

// Features generates a feature vector given a method name and the query
func (f *MethodFeaturer) Features(name string, query *precompute.QueryStats, targets *precompute.TargetStats) []float64 {
	var feats []float64
	for _, featurer := range f.Featurers {
		feats = append(feats, featurer.Features(name, query, targets))
	}
	return feats
}

// Labels returns the feature names
func (f *MethodFeaturer) Labels() []string {
	var labels []string
	for _, featurer := range f.Featurers {
		labels = append(labels, featurer.Label())
	}
	return labels
}

// --

// matchFeaturer returns the number of exact matched tokens in the query and
// in the method name.
type matchFeaturer struct {
	Matchers map[string]*KeywordMatcher
	Name     string
}

// newMatchFeaturer returns a pointer to a new matchFeaturer object.
func newMatchFeaturer(names map[string][]string, label string) *matchFeaturer {
	matchers := make(map[string]*KeywordMatcher)
	for id, syns := range names {
		var tokens []string
		for _, s := range syns {
			parts := strings.Split(s, ".")
			tokens = append(tokens, parts[len(parts)-1])
		}
		matchers[id] = newKeywordMatcher(text.Uniquify(text.Lower(tokens)))
	}
	return &matchFeaturer{
		Matchers: matchers,
		Name:     label,
	}
}

// Features returns the number of exact matches for tokens in the query
// and the method name.
func (f *matchFeaturer) Features(name string, query *precompute.QueryStats, targets *precompute.TargetStats) float64 {
	if matcher, found := f.Matchers[name]; found {
		feat, matched := matcher.match(query.UnmatchedTokens)
		query.UnmatchedTokens = filter(query.UnmatchedTokens, matched)
		return feat
	}
	return 0
}

// Label returns the name of the feature.
func (f *matchFeaturer) Label() string {
	return f.Name
}

// --

// fuzzyMatchFeaturer finds how many fuzzy-matches are between the method name
// and the query.
type fuzzyMatchFeaturer struct {
	Matchers map[string]*FuzzyKeywordMatcher
	Name     string
}

// newFuzzyMatchFeaturer returns a pointer to a new fuzzyMatchFeaturer object.
func newFuzzyMatchFeaturer(names map[string][]string, label string) *fuzzyMatchFeaturer {
	matchers := make(map[string]*FuzzyKeywordMatcher)
	for id, syns := range names {
		var tokens []string
		for _, s := range syns {
			parts := strings.Split(s, ".")
			tokens = append(tokens, parts[len(parts)-1])
		}
		matchers[id] = newFuzzyKeywordMatcher(text.Uniquify(text.Lower(tokens)))
	}
	return &fuzzyMatchFeaturer{
		Matchers: matchers,
		Name:     label,
	}
}

// Features returns the number of fuzzy matching between tokens in the query
// and the function name.
func (f *fuzzyMatchFeaturer) Features(name string, query *precompute.QueryStats, targets *precompute.TargetStats) float64 {
	if matcher, found := f.Matchers[name]; found {
		// remove package name from query tokens
		feat, matched := matcher.match(query.UnmatchedTokens)
		query.UnmatchedTokens = filter(query.UnmatchedTokens, matched)
		return feat
	}
	return 0

}

// Label returns the name of the feature.
func (f *fuzzyMatchFeaturer) Label() string {
	return f.Name
}

// --

// logPriorFeaturer returns the prior probability of a function name
type logPriorFeaturer struct {
	LogPrior map[string]float64
	Name     string
}

// Features returns the log prior probability of the given method name.
func (f *logPriorFeaturer) Features(name string, query *precompute.QueryStats, targets *precompute.TargetStats) float64 {
	logProb, found := f.LogPrior[name]
	if !found {
		return 0
	}
	return math.Exp(logProb)
}

// Label returns the name of the feature.
func (f *logPriorFeaturer) Label() string {
	return f.Name
}

// --

// tfidfFeaturer computes tfidf features
type tfidfFeaturer struct {
	Scorer *tfidf.Scorer
	Name   string
}

// newTFIDFFeaturer returns a pointer to a new tfidfFeaturer object.
func newTFIDFFeaturer(corpus map[string][]string, label string) *tfidfFeaturer {
	return &tfidfFeaturer{
		Scorer: tfidf.TrainScorer(corpus, false, text.TokenizeNoCamel),
		Name:   label,
	}
}

// Features returns the cosine distance between the query and the method in the
// tfidf space.
func (f *tfidfFeaturer) Features(name string, query *precompute.QueryStats, targets *precompute.TargetStats) float64 {
	if query.TFCounter == nil {
		// get the tfidf length of the query
		tfCorpus := make(map[string]int)
		for _, tok := range query.StemmedTokens {
			tfCorpus[tok]++
		}
		query.TFCounter = tfidf.TrainTFCounter(true, tfCorpus)
		query.TFIDFNorm = f.Scorer.ComputeNorm(query.TFCounter)
	}
	return f.Scorer.TFIDFScoreWithTFCounter(query.TFCounter, query.TFIDFNorm, name)
}

// Label returns the name of the feature.
func (f *tfidfFeaturer) Label() string {
	return f.Name
}

// --

// lmFeaturer compute language model based features
type lmFeaturer struct {
	Scorer *LmMethodPredictor
	Name   string
}

// newLMFeaturer returns a pointer to a new lmFeaturer object.
func newLMFeaturer(corpus map[string][]string, label string) *lmFeaturer {
	data := make(map[string]*MethodTrainingData)
	for m, docs := range corpus {
		tokens := make([]string, 0, 10000)
		for _, doc := range docs {
			tokens = append(tokens, lmProcessor.Apply(text.TokenizeNoCamel(doc))...)
		}
		data[m] = &MethodTrainingData{
			LogPrior: 0,
			Data:     tokens,
		}
	}
	lm := NewLmMethodPredictor(backgroundLMProb, data)
	return &lmFeaturer{
		Scorer: lm,
		Name:   label,
	}
}

// Features returns the posterior language model probability for the given method name for the query.
func (f *lmFeaturer) Features(name string, query *precompute.QueryStats, target *precompute.TargetStats) float64 {
	if target.LMScores[f.Name] == nil {
		max := math.Inf(-1)
		target.LMScores[f.Name] = make(map[string]float64)
		for _, c := range target.Candidates {
			p := f.Scorer.MethodLogLikelihood(c, query.StemmedTokens)
			target.LMScores[f.Name][c] = p
			if p > max {
				max = p
			}
		}
		for c := range target.LMScores[f.Name] {
			target.LMScores[f.Name][c] = math.Exp(target.LMScores[f.Name][c] - max)
		}
	}
	if prob, found := target.LMScores[f.Name][name]; found {
		return prob
	}
	return 0
}

// Label returns the name of the feature.
func (f *lmFeaturer) Label() string {
	return f.Name
}

// --

// wordToMethodFeaturer builds the language model
type wordToMethodFeaturer struct {
	Word2MethodModel map[string][]float64
	MethodToID       map[string]int
	MethodToSelector map[string]string
	Name             string
}

// newWordToMethodFeaturer(soCorpus, docCorpus, curationCorpus, "word_to_method"))
func newWordToMethodFeaturer(soCorpus, docCorpus, curationCorpus map[string][]string, label string) *wordToMethodFeaturer {
	methods, wordCounts := buildWordCount(soCorpus, docCorpus, curationCorpus)

	methodToID := make(map[string]int)
	methodToSelector := make(map[string]string)
	for i, m := range methods {
		methodToID[m] = i
		parts := strings.Split(m, ".")
		methodToSelector[m] = strings.ToLower(parts[len(parts)-1])
	}

	var ptr int
	for ptr = 0; ptr < len(wordCounts); ptr++ {
		if wordCounts[ptr].count > minWordCount {
			break
		}
	}

	wordCounts = wordCounts[ptr:]

	// get top words
	wordToMethods := make(map[string][]float64)
	for _, wc := range wordCounts {
		wordToMethods[wc.word] = make([]float64, len(methods))
	}

	countWordToMethods(wordToMethods, methodToID, soCorpus, docCorpus, curationCorpus)

	for w, counts := range wordToMethods {
		var total float64
		for i := range counts {
			// add one smoothing
			counts[i]++
			total += counts[i]
		}
		for i := range counts {
			counts[i] = math.Log(counts[i] / total)
		}
		wordToMethods[w] = counts
	}
	return &wordToMethodFeaturer{
		Word2MethodModel: wordToMethods,
		MethodToID:       methodToID,
		Name:             label,
	}
}

// Features returns the feature defined by wordToMethodFeaturer
func (wf *wordToMethodFeaturer) Features(name string, query *precompute.QueryStats, target *precompute.TargetStats) float64 {
	if target.WordToMethodScores == nil {
		scores := make(map[string]float64)
		var unmatchedWords []string

		for _, c := range target.Candidates {
			id, found := wf.MethodToID[c]
			if !found {
				scores[c] = math.Inf(-1)
				continue
			}
			var score float64
			for _, t := range query.StemmedTokens {
				if prob, found := wf.Word2MethodModel[t]; found {
					score += prob[id]
				} else {
					unmatchedWords = append(unmatchedWords, t)
				}
			}
			scores[c] = score
		}

		// check whether any unmatched words correspond to selector names
		for _, w := range unmatchedWords {
			count := make([]float64, len(wf.MethodToID))
			var total float64
			for _, orig := range query.StemmedToOriginal[w] {
				for method, id := range wf.MethodToID {
					if orig == wf.MethodToSelector[method] {
						count[id]++
						total++
					}
				}
			}
			if total > 0 {
				for method, id := range wf.MethodToID {
					if count[id] == 0 {
						scores[method] += minLogProb
					} else {
						scores[method] += math.Log(count[id] / total)
					}
				}
			}
		}
		max := math.Inf(-1)
		for _, score := range scores {
			if score > max {
				max = score
			}
		}

		for c, s := range scores {
			if max == math.Inf(-1) {
				scores[c] = 0
			} else {
				scores[c] = math.Exp(s - max)
			}
		}
		target.WordToMethodScores = scores
	}
	if prob, found := target.WordToMethodScores[name]; found {
		return prob
	}
	return 0
}

// Label returns the name of the featurer
func (wf *wordToMethodFeaturer) Label() string {
	return wf.Name
}

// --

// isPackageFeaturer checks whether the name corresponds to the package name
type isPackageFeaturer struct {
	Package string
	Name    string
}

// Features checks whether the method name corresponds to the package name
// and whether the query contains package name
func (f *isPackageFeaturer) Features(name string, query *precompute.QueryStats, target *precompute.TargetStats) float64 {
	if name == f.Package && !strings.Contains(strings.ToLower(query.Raw), f.Package) {
		return 1.0
	}
	return 0.0
}

// Label returns the name of the featurer
func (f *isPackageFeaturer) Label() string {
	return f.Name
}

// --

// depthFeaturer returns the depth of the method name in the import graph (with root being
// at the package node).
type depthFeaturer struct {
	Name string
}

// newDepthFeaturer returns a pointer to a new depthFeaturer object.
func newDepthFeaturer(label string) *depthFeaturer {
	return &depthFeaturer{
		Name: label,
	}
}

// Features returns the depth of the method name in the import graph
func (d *depthFeaturer) Features(name string, query *precompute.QueryStats, target *precompute.TargetStats) float64 {
	return 1.0 / float64(len(strings.Split(name, ".")))
}

// Label returns the name of the featurer
func (d *depthFeaturer) Label() string {
	return d.Name
}
