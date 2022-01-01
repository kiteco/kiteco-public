package pythonranker

import (
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker/internal/precompute"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/languagemodel"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	wordVecLen         = 13003
	defaultPseudoCount = 0.001
)

func init() {
	gob.Register(&MatchPredictor{})
	gob.Register(&LmMethodPredictor{})
	gob.Register(&matchFeaturer{})
	gob.Register(&fuzzyMatchFeaturer{})
	gob.Register(&logPriorFeaturer{})
	gob.Register(&tfidfFeaturer{})
	gob.Register(&lmFeaturer{})
	gob.Register(&wordToMethodFeaturer{})
	gob.Register(&isPackageFeaturer{})
	gob.Register(&depthFeaturer{})
}

// Predictor defines the interface that any implementation of a method prediction model
// must satisfy.
type Predictor interface {
	Scores([]string) []*MethodScore
}

// MethodPredictor consists of different prediction models and weights on these
// various models.
type MethodPredictor struct {
	Models  []Predictor
	Weights []float64
}

// NewMethodPredictor takes in a pretrained MatchPredictor and LmMethodPredictor and the weights
// for each of the two predictors and returns a pointer to a new MethodPredictor object.
func NewMethodPredictor(matcher *MatchPredictor, lm *LmMethodPredictor, weights []float64) *MethodPredictor {
	if len(weights) != 2 {
		log.Fatal("must specify weights for matching scores and lm scores, e.g., [0.5, 0.5]")
	}
	return &MethodPredictor{
		Models:  []Predictor{matcher, lm},
		Weights: weights,
	}
}

// PredictTopNSels returns the top n most likely selector names given the query tokens.
func (mp *MethodPredictor) PredictTopNSels(tokens []string, n int) []string {
	data := mp.Predict(tokens)
	// compute entropy of the scores
	var topMethods []string
	for i := 0; i < n && i < len(data); i++ {
		topMethods = append(topMethods, data[i].Name)
	}
	return topMethods
}

// Predict returns how likely the tokens refer to each method.
func (mp *MethodPredictor) Predict(tokens []string) []*MethodScore {
	var combinedScores []*MethodScore
	scoreMap := make(map[string]*MethodScore)
	for i, m := range mp.Models {
		scores := m.Scores(tokens)
		for _, ms := range scores {
			var score *MethodScore
			var exists bool
			if score, exists = scoreMap[ms.Name]; !exists {
				score = &MethodScore{
					Name: ms.Name,
				}
				scoreMap[ms.Name] = score
				combinedScores = append(combinedScores, score)
			}
			score.Score += mp.Weights[i] * ms.Score
		}
	}
	sort.Sort(sort.Reverse(byScore(combinedScores)))
	return combinedScores
}

// MethodScore represents the method name and the score the method gets from
// the model.
type MethodScore struct {
	Name  string
	Score float64
}

type byScore []*MethodScore

func (bs byScore) Len() int           { return len(bs) }
func (bs byScore) Less(i, j int) bool { return bs[i].Score < bs[j].Score }
func (bs byScore) Swap(i, j int)      { bs[j], bs[i] = bs[i], bs[j] }

// MatchPredictor implements a predictor by checking whether any tokens
// in the query string matches with any function / class method names
// in a package.
type MatchPredictor struct {
	Names []string
}

// NewMatchPredictor returns a pointer to a
func NewMatchPredictor(names []string) *MatchPredictor {
	return &MatchPredictor{
		Names: names,
	}
}

// Scores returns the score that each method gets for the given query tokens.
// The score is computed as follows.
// p(m|query) \prop (I(m in query) + alpha / (\sum_m I(m in query) + alpha))
// where m stands for the selector name, and I(.) is 1 if the statement is true;
// otherwise, it's 0.
func (mp *MatchPredictor) Scores(tokens []string) []*MethodScore {
	pseudoCount := defaultPseudoCount
	if len(mp.Names) > 0 {
		if pseudoCount > 1.0/float64(len(mp.Names)) {
			pseudoCount = 1.0 / float64(len(mp.Names))
		}
	}

	seenTokens := make(map[string]struct{})
	for _, tok := range tokens {
		seenTokens[tok] = struct{}{}
	}

	counts := make(map[string]float64)

	var total float64
	for _, m := range mp.Names {
		counts[m] = pseudoCount
		if _, seen := seenTokens[m]; seen {
			counts[m]++
		}
		total += counts[m]
	}

	var scores []*MethodScore
	for _, m := range mp.Names {
		score := counts[m] / total
		scores = append(scores, &MethodScore{
			Name:  m,
			Score: score,
		})
	}
	return scores
}

// LmMethodPredictor implements the following graphical model
// f     i
//  \   /
//    w
// where f stands for a method name, and i stands for whether
// a background (universal model) was used to generate the word w.
// The graphical model encodes the following probability distribution.
// p(w|f) = \sum_i p(w, i | f) = p(w, i = 0 | f) + p(w, i = 1 | f)
//        = p(i = 0) * p(w | f) + p(i = 1) * p(w | background)
// As a result, to infer p(f|w), we do
// p(f|w) = p(w|f) * p(f) / (\sum_f p(w|f) * p(f))
// The parameters of this model are then:
// 1) p(i)
// 2) p(f)
// 3) p(w|f)
// 4) p(w|background)
// We set p(i) by using cross validation and use github stats to estimate p(f).
// We use doc / stack overflow data to infer p(w|f) and p(w|background).
// Note that if UseBackground = 0.0, then the model will reduce to a
// naive unigram language model.
type LmMethodPredictor struct {
	useBackground    float64
	notUseBackground float64

	LogBackgroundProb    float64
	LogNotBackgroundProb float64

	MethodNames    []string
	MethodLogPrior []float64
	MethodLMs      []*languagemodel.UnigramLanguageModel
	BackgroundLM   *languagemodel.UnigramLanguageModel
}

// NewLmMethodPredictor returns a pointer to a new LmMethodPredictor object.
func NewLmMethodPredictor(bgProb float64, data map[string]*MethodTrainingData) *LmMethodPredictor {
	predictor := LmMethodPredictor{
		useBackground:        bgProb,
		notUseBackground:     1 - bgProb,
		LogBackgroundProb:    math.Log(bgProb),
		LogNotBackgroundProb: math.Log(1 - bgProb),
	}
	trainingTokens := make([]string, 0, 10000)
	for m, d := range data {
		lm := languagemodel.TrainUnigramLanguageModel(d.Data, wordVecLen)
		predictor.MethodLMs = append(predictor.MethodLMs, lm)
		predictor.MethodLogPrior = append(predictor.MethodLogPrior, d.LogPrior)
		predictor.MethodNames = append(predictor.MethodNames, m)

		trainingTokens = append(trainingTokens, d.Data...)
	}
	predictor.BackgroundLM = languagemodel.TrainUnigramLanguageModel(trainingTokens, wordVecLen)
	return &predictor
}

func index(candidates []string, m string) int {
	for i := range candidates {
		if candidates[i] == m {
			return i
		}
	}
	return -1
}

// SetLogPrior sets the log prior on the methods. This is useful
// for doing cross validation.
func (md *LmMethodPredictor) SetLogPrior(prior map[string]float64) error {
	for m, p := range prior {
		id := index(md.MethodNames, m)
		if id < 0 {
			return fmt.Errorf("cannot find %s in the list of selector names", m)
		}
		md.MethodLogPrior[id] = p
	}
	return nil
}

// SetBackgroundWeight sets how much weight should we put on the background
// language model. This is used during cross validation.
func (md *LmMethodPredictor) SetBackgroundWeight(weight float64) {
	md.useBackground = weight
	md.notUseBackground = 1 - weight
	md.LogBackgroundProb = math.Log(weight)
	md.LogNotBackgroundProb = math.Log(1 - weight)
}

// Scores computes p(f|w) for each method given w.
func (md *LmMethodPredictor) Scores(tokens []string) []*MethodScore {
	var scores []float64
	var results []*MethodScore
	for i, prior := range md.MethodLogPrior {
		prob := prior
		for _, tok := range tokens {
			p := logSumExp([]float64{md.LogBackgroundProb + md.BackgroundLM.LogLikelihood([]string{tok}),
				md.LogNotBackgroundProb + md.MethodLMs[i].LogLikelihood([]string{tok})})
			prob += p
		}
		results = append(results, &MethodScore{
			Name:  md.MethodNames[i],
			Score: prob,
		})
		scores = append(scores, prob)
	}
	total := logSumExp(scores)
	for _, m := range results {
		m.Score -= total
		m.Score = math.Exp(m.Score)
	}
	return results
}

// MethodLogLikelihood computes the probability of the given query tokens
func (md *LmMethodPredictor) MethodLogLikelihood(method string, tokens []string) float64 {
	var prob float64
	id := index(md.MethodNames, method)
	if id < 0 {
		return minLogProb
	}
	for _, tok := range tokens {
		background := md.LogBackgroundProb + md.BackgroundLM.LogLikelihood([]string{tok})
		specific := md.LogNotBackgroundProb + md.MethodLMs[id].LogLikelihood([]string{tok})
		prob += logSumExp([]float64{background, specific})
	}
	return prob
}

// --

// MethodTrainingData holds the training data for a method.
// This data struct is temporarily placed here util the searcher for
// the doc corpus is merged in.
type MethodTrainingData struct {
	Data     []string
	LogPrior float64
}

// --

// MethodRanker contains the featurers for each package and the pre-trained model.
type MethodRanker struct {
	featurers map[string]*MethodFeaturer
	model     *ranking.Ranker
}

// Features converts the list of candidate method names to a list of data points used
// for ranking.
func (mr *MethodRanker) Features(query, packageName string, candidates []string) ([]*ranking.DataPoint, error) {
	featurer, found := mr.featurers[packageName]
	if !found {
		return nil, fmt.Errorf("cannot find a featurer for package: %s", packageName)
	}

	queryStats := precompute.NewQueryStats(query, packageName)
	targetStats := precompute.NewTargetStats(candidates)

	var data []*ranking.DataPoint
	for _, c := range candidates {
		queryStats.Reset()
		feats := featurer.Features(c, queryStats, targetStats)
		data = append(data, &ranking.DataPoint{
			Name:     c,
			Features: feats,
		})
	}
	return data, nil
}

func (mr *MethodRanker) rank(query, packageName string, candidates []string) ([]*ranking.DataPoint, error) {
	data, err := mr.Features(query, packageName, candidates)
	if err != nil {
		return nil, err
	}
	mr.model.Rank(data)
	return data, nil
}

// Rank ranks the given candidate methods for the given query
func (mr *MethodRanker) Rank(query, packageName string, candidates []string) ([]string, error) {
	data, err := mr.rank(query, packageName, candidates)
	if err != nil {
		return nil, err
	}
	var sortedMethods []string
	for _, d := range data {
		sortedMethods = append(sortedMethods, d.Name)
	}
	return sortedMethods, nil
}

// FindWithScore finds the entities that match with the given name and returns their scores.
func (mr *MethodRanker) FindWithScore(query, packageName string, candidates []string) ([]*ranking.DataPoint, error) {
	_, found := mr.featurers[packageName]
	if !found {
		return nil, fmt.Errorf("cannot find a featurer for package: %s", packageName)
	}

	queryTokens := make(map[string]struct{})
	for _, tok := range text.Lower(strings.Split(query, " ")) {
		queryTokens[tok] = struct{}{}
	}

	var data []*ranking.DataPoint
	for _, c := range candidates {
		tokens := text.Lower(strings.Split(c, "."))
		if _, match := queryTokens[tokens[len(tokens)-1]]; match {
			data = append(data, &ranking.DataPoint{
				Name:  c,
				Score: 1.0 / float64(len(tokens)+1),
			})
			break
		}
	}
	sort.Sort(sort.Reverse(ranking.ByScore(data)))
	return data, nil
}

// Find finds the entities that match with the given name
func (mr *MethodRanker) Find(query, packageName string, candidates []string) ([]string, error) {
	data, err := mr.FindWithScore(query, packageName, candidates)
	if err != nil {
		return nil, err
	}

	var sortedMethods []string
	for _, d := range data {
		sortedMethods = append(sortedMethods, d.Name)
	}

	return sortedMethods, nil
}

// RankWithScores ranks the given candidate methods for the given query and returns the
// associated scores.
func (mr *MethodRanker) RankWithScores(query, packageName string, candidates []string) ([]*ranking.DataPoint, error) {
	return mr.rank(query, packageName, candidates)
}

// NewMethodRankerFromFile returns a pointer to a new Ranker for
// ranking package methods. It also loads the featurer used to generate
// features.
func NewMethodRankerFromFile(modelPath, featurerPath string) (*MethodRanker, error) {
	in, err := fileutil.NewCachedReader(modelPath)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	model, err := ranking.NewRankerFromJSON(in)
	if err != nil {
		return nil, err
	}

	fin, err := fileutil.NewCachedReader(featurerPath)
	if err != nil {
		return nil, err
	}

	featurers, err := NewMethodFeaturerFromFile(fin)
	if err != nil {
		return nil, err
	}

	return &MethodRanker{
		model:     model,
		featurers: featurers,
	}, nil
}

// PackageRanker contains a featurer that converts a query into a list of
// feature vectors, of which each is a function of the query and a package.
// It also contains a ranker that ranks the packages accordingly to
// how likely the query refers to the packages.
type PackageRanker struct {
	featurer *PackageFeaturer
	model    *ranking.Ranker
}

// NewPackageRankerFromJSON returns a pointer to a new Ranker object by loading
// both the ranking model and the featurer from a json file.
func NewPackageRankerFromJSON(modelPath, featurerPath string) (*PackageRanker, error) {
	in, err := fileutil.NewCachedReader(modelPath)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	model, err := ranking.NewRankerFromJSON(in)
	if err != nil {
		return nil, err
	}

	fin, err := fileutil.NewCachedReader(featurerPath)
	if err != nil {
		return nil, err
	}
	featurer, err := NewPackageFeaturerFromJSON(fin)
	if err != nil {
		return nil, err
	}

	err = checkModelFeaturerConsistency(model, featurer)
	if err != nil {
		return nil, err
	}

	return &PackageRanker{
		model:    model,
		featurer: featurer,
	}, nil
}

func checkModelFeaturerConsistency(model *ranking.Ranker, featurer *PackageFeaturer) error {
	modelLabels := model.FeatureLabels
	featurerLabels := featurer.Labels()

	if len(modelLabels) != len(featurerLabels) {
		return fmt.Errorf("len of features used in model and features don't match: %d v.s. %d", len(modelLabels), len(featurerLabels))
	}
	for i, ml := range modelLabels {
		if ml != featurerLabels[i] {
			return fmt.Errorf("model feature %s and featurer %s are not consistent", ml, featurerLabels[i])
		}
	}
	return nil
}

// Find finds packages that contain selector names that match with the query.
func (p *PackageRanker) Find(query string) []*ranking.DataPoint {
	data := p.featurer.Find(query)
	sort.Sort(sort.Reverse(ranking.ByScore(data)))
	return data
}

// Rank ranks how likely the given query refers to each possible package
// in the doc corpus. The returned results are sorted based on their score.
// The package name can be accessed by doing DataPoint.Name.
func (p *PackageRanker) Rank(query string) []*ranking.DataPoint {
	data := p.featurer.Features(query)
	p.model.Rank(data)
	sort.Sort(sort.Reverse(ranking.ByScore(data)))
	return data
}

// PackagePredictor contains a featurer that converts a query into a set of
// feature vectors and a classifier that outputs how likely the query
// refers to each package.
type PackagePredictor struct {
	featurer   *PackageFeaturer
	classifier *ranking.BinaryClassifier
}

// Predict predicts how likely the given query refers to each possible package
// in the doc corpus. The returned results are sorted based on their score.
// The package name can be accessed by doing DataPoint.Name.
func (p *PackagePredictor) Predict(query string) []*ranking.DataPoint {
	data := p.featurer.Features(query)
	for _, d := range data {
		d.Score = p.classifier.PredictProba(d.Features)
	}
	sort.Sort(sort.Reverse(ranking.ByScore(data)))
	return data
}

// NewPackagePredictorFromFile loads the featurer and the model to construct
// an new PackagePredictor from files.
func NewPackagePredictorFromFile(root string) (*PackagePredictor, error) {
	featurerPath := fileutil.Join(root, "featurer.json")
	feat, err := fileutil.NewCachedReader(featurerPath)
	if err != nil {
		return nil, err
	}
	defer feat.Close()

	featurer, err := NewPackageFeaturerFromJSON(feat)
	if err != nil {
		return nil, err
	}

	modelPath := fileutil.Join(root, "model.json")
	fmodel, err := fileutil.NewCachedReader(modelPath)
	if err != nil {
		return nil, err
	}
	defer fmodel.Close()

	classifier, err := ranking.NewBinaryClassifierFromJSON(fmodel)
	if err != nil {
		return nil, err
	}

	return &PackagePredictor{
		featurer:   featurer,
		classifier: classifier,
	}, nil
}

// Candidates returns the list of packages considered by the ranker.
func (p *PackageRanker) Candidates() []string {
	var candidates []string
	for n := range p.featurer.PackageList {
		candidates = append(candidates, n)
	}
	return candidates
}
