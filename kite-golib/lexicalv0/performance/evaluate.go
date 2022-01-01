package performance

import (
	"math/rand"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// Measurement is the category of a metric
type Measurement string

const (
	//Lexical ...
	Lexical Measurement = "lexical"
	// Word ...
	Word Measurement = "word"
	// String ...
	String Measurement = "string"
	// StringCharValueAdded ...
	StringCharValueAdded Measurement = "string_char_value_added"
	// Series ...
	Series Measurement = "series"
	// TokenValueAdded ...
	TokenValueAdded Measurement = "token_value_added"
	// CharValueAdded ...
	CharValueAdded Measurement = "char_value_added"
	// CorrectTokensAhead ...
	CorrectTokensAhead Measurement = "correct_tokens_ahead"
)

// Metric is for any measurement
type Metric struct {
	Accurate float64
	InTopK   float64
	Err      error
}

// Summary is aggregated metrics
type Summary map[Measurement][]Metric

// EncodingLength ...
type EncodingLength struct {
	Positive []int
	Negative []int
}

// Evaluator ...
type Evaluator struct {
	Filename        string
	Encoder         *lexicalv0.FileEncoder
	Predictor       predict.Predictor
	Search          predict.SearchConfig
	SeriesLength    int
	TopN            int
	Rand            *rand.Rand
	RandomSeed      int64
	Measurements    map[Measurement]float64
	NumTopPopular   int
	MinMuddle       int
	UnmuddledWindow int

	encLen                 EncodingLength
	evalSummary            Summary
	totalEntropy           float64
	totalWords             int
	groupedTokenValueAdded map[int][]Metric
	report                 string
}

func (e *Evaluator) newPredictInputs(tokens []lexer.Token, prefix string, seed int64) predict.Inputs {
	return predict.Inputs{
		FilePath:       e.Filename,
		Prefix:         prefix,
		Tokens:         e.maybeMuddle(tokens),
		CursorTokenIdx: len(tokens),
		RandomSeed:     seed,
	}
}

// ValueAdded returns the prefix string needed for the model to predict the rest, TokenValueAdded and CharValueAdded
func (e *Evaluator) ValueAdded(context []lexer.Token, window []lexer.Token) (string, Metric, Metric, error) {
	var tokenValueAdded Metric
	var charValueAdded Metric
	var site []string
	for _, w := range window {
		site = append(site, getString(e.Encoder.Lexer, w))
	}
	tokenLevelValueAdded := func(i, j int) float64 {
		givenTokens := float64(i) + (float64(j) / float64(len(site[i])))
		return 1 - (givenTokens / float64(len(site)))
	}

	charLevelValueAdded := func(i, j int) float64 {
		var totalLength int
		var givenLength int
		for id, s := range site {
			totalLength += len(s)
			if id < i {
				givenLength += len(s)
			}
		}
		givenLength += j
		return 1 - (float64(givenLength) / float64(totalLength))
	}

	prefixString := func(i, j int) string {
		var prefix []string
		for id := 0; id < i; id++ {
			prefix = append(prefix, site[id])
		}
		prefix = append(prefix, site[i][:j])

		return strings.Join(prefix, " ")
	}

	prefix := "KITE_NO_LUCK"

	// copy the search config so we can update the depth in a go routine safe way
	config := e.Search

iterate:
	for i, s := range site {
		// deep copy of tokens since we may modify this below
		updatedContext := append([]lexer.Token{}, context...)
		updatedContext = append(updatedContext, window[:i]...)

		for j := range s {
			currentPrefix := s[:j]

			updatedLabel := e.Encoder.EncodeTokens(window[i:])

			inputs := e.newPredictInputs(updatedContext, currentPrefix, e.RandomSeed)
			if originalSubs, ok := e.Encoder.Lexer.ShouldBPEEncode(window[i]); ok {
				// check if we need to compute an updated prefix and label
				// based on if there are multiple subtokens present
				// for the current token. This is mostly for JS strings,
				// SEE: predict/context.go
				// TODO: this is pretty hacky, we split the literal for the token at i and put part
				// of it in the context as newTok and the other part into the label
				newTok := lexer.Token{
					Start: window[i].Start,
					End:   window[i].Start + len(currentPrefix),
					Lit:   currentPrefix,
				}
				if currentSubs, _ := e.Encoder.Lexer.ShouldBPEEncode(newTok); len(currentSubs) > 1 {
					if len(originalSubs) < len(currentSubs) {
						return prefix, Metric{}, Metric{}, errors.New("error parsing prefix correctly")
					}

					// add the new token to the context, and put the cursor on this new token,
					// then predict/context.go/buildLeftContextAndSetPrefix will do the right
					// thing with this partial token and update the prefix accordingly
					updatedContext = append(updatedContext, newTok)
					inputs = predict.Inputs{
						Prefix:         currentPrefix,
						Tokens:         updatedContext,
						CursorTokenIdx: len(updatedContext) - 1,
						RandomSeed:     e.RandomSeed,
					}

					// update the label
					updatedLabel = e.Encoder.EncodeSubtokens(originalSubs[len(currentSubs)-1:])
					if i < len(site)-1 {
						updatedLabel = append(updatedLabel, e.Encoder.EncodeTokens(window[i+1:])...)
					}
				}
			}

			depth := len(updatedLabel)
			if depth >= e.Predictor.GetHParams().NumPredictionSlots {
				continue
			}
			config.Depth = depth
			inputs.SearchConfig = config // override default search config

			res, err := e.Predictor.Predict(kitectx.Background(), inputs)
			if err != nil {
				return prefix, Metric{}, Metric{}, errors.Wrapf(err, "prediction error")
			}
			predicted := res.Preds

			metric := matchSeries(predicted, updatedLabel, e.TopN)

			if metric.Accurate == 1.0 {
				tokenValueAdded.Accurate = tokenLevelValueAdded(i, j)
				charValueAdded.Accurate = charLevelValueAdded(i, j)
				if tokenValueAdded.InTopK == 0 {
					tokenValueAdded.InTopK = tokenValueAdded.Accurate
				}
				if charValueAdded.InTopK == 0 {
					charValueAdded.InTopK = charValueAdded.Accurate
				}
				prefix = prefixString(i, j)
				break iterate
			}
			if metric.InTopK == 1.0 {
				tokenValueAdded.InTopK = tokenLevelValueAdded(i, j)
				charValueAdded.InTopK = charLevelValueAdded(i, j)
			}
		}
	}

	return prefix, tokenValueAdded, charValueAdded, nil
}

func (e *Evaluator) accuracy(contextTokens []lexer.Token, label []lexer.Token) (Metric, int, error) {
	correct := e.Encoder.EncodeTokens(label)
	config := e.Search
	// enough depth to predict ground truth
	config.Depth = len(correct)
	if config.Depth > e.Predictor.GetHParams().NumPredictionSlots {
		return Metric{Err: errDepthExceedsSlots}, 0, nil
	}

	in := e.newPredictInputs(contextTokens, "", e.RandomSeed)
	in.SearchConfig = config // override default search config
	res, err := e.Predictor.Predict(kitectx.Background(), in)
	if err != nil {
		return Metric{}, 0, errors.Cause(err)
	}
	series := res.Preds

	metric := matchSeries(series, correct, e.TopN)
	return metric, len(correct), nil
}

func (e *Evaluator) metrics(sites PredictionSites) error {
	summary := make(Summary)

	for _, site := range sites[Series] {
		metric, _, err := e.accuracy(site.Tokens[:site.Idx], site.Tokens[site.Idx:site.Idx+e.SeriesLength])
		if err != nil {
			return err
		}
		summary[Series] = append(summary[Series], metric)
	}

	for _, site := range sites[TokenValueAdded] {
		window := site.Tokens[site.Idx : site.Idx+e.SeriesLength]

		_, tokenMetric, charMetric, err := e.ValueAdded(site.Tokens[:site.Idx], window)
		if err != nil {
			return errors.Wrapf(err, "error computing value added")
		}
		summary[TokenValueAdded] = append(summary[TokenValueAdded], tokenMetric)
		summary[CharValueAdded] = append(summary[CharValueAdded], charMetric)
		group := numIdents(e.Encoder.Lexer, window)
		e.groupedTokenValueAdded[group] = append(e.groupedTokenValueAdded[group], tokenMetric)
	}

	for _, site := range sites[Word] {
		tok := site.Tokens[site.Idx]
		metric, trueDepth, err := e.accuracy(site.Tokens[:site.Idx], []lexer.Token{tok})
		if err != nil {
			return errors.Wrapf(err, "error doing prediction")
		}

		if metric.Err == nil {
			// Keep track of the encoding length
			if metric.InTopK > 0 {
				e.encLen.Positive = append(e.encLen.Positive, trueDepth)
			} else {
				e.encLen.Negative = append(e.encLen.Negative, trueDepth)
			}
		}
		summary[Word] = append(summary[Word], metric)
	}

	for _, site := range sites[String] {
		tok := site.Tokens[site.Idx]
		metric, trueDepth, err := e.accuracy(site.Tokens[:site.Idx], []lexer.Token{tok})
		if err != nil {
			return err
		}
		if metric.Err == nil {
			// Keep track of the encoding length
			if metric.InTopK > 0 {
				e.encLen.Positive = append(e.encLen.Positive, trueDepth)
			} else {
				e.encLen.Negative = append(e.encLen.Negative, trueDepth)
			}
		}
		summary[String] = append(summary[String], metric)
	}

	for _, site := range sites[StringCharValueAdded] {
		tok := site.Tokens[site.Idx]
		_, _, cva, err := e.ValueAdded(site.Tokens[:site.Idx], []lexer.Token{tok})
		if err != nil {
			return errors.Cause(err)
		}
		summary[StringCharValueAdded] = append(summary[StringCharValueAdded], cva)
	}

	for _, site := range sites[Lexical] {
		tok := site.Tokens[site.Idx]
		metric, _, err := e.accuracy(site.Tokens[:site.Idx], []lexer.Token{tok})
		if err != nil {
			return err
		}
		summary[Lexical] = append(summary[Lexical], metric)
	}
	e.evalSummary.append(summary)
	return nil
}

func (e *Evaluator) correctTokensAhead(sites PredictionSites, maxCorrectAhead int) error {
	summary := make(Summary)

	for _, site := range sites[CorrectTokensAhead] {
		contextTokens := site.Tokens[:site.Idx]
		labelTokens := site.Tokens[site.Idx : site.Idx+maxCorrectAhead]
		depthNeeded := len(e.Encoder.EncodeTokens(labelTokens))
		for j := 1; j < maxCorrectAhead; j++ {
			if depthNeeded <= e.Predictor.GetHParams().NumPredictionSlots {
				// NOTE: if we want to remove this dependence we can make an alternate
				// code path in the predictor that doesn't use the PRM and then we
				// can search as long as we want.
				break
			}
			labelTokens = site.Tokens[site.Idx : site.Idx+maxCorrectAhead-j]
			depthNeeded = len(e.Encoder.EncodeTokens(labelTokens))
		}

		search := e.Search
		search.Depth = depthNeeded
		if search.Depth > e.Predictor.GetHParams().NumPredictionSlots {
			continue
		}

		in := e.newPredictInputs(contextTokens, "", e.RandomSeed)
		in.SearchConfig = search

		// use predict chan to get all intermediate results
		var allPreds []predict.Predicted
		predChan, errChan := e.Predictor.PredictChan(kitectx.Background(), in)
		for pred := range predChan {
			allPreds = append(allPreds, pred)
		}

		if err := <-errChan; err != nil {
			return errors.Cause(err)
		}

		byTokenLen := make(map[int][]predict.Predicted)
		for _, pred := range allPreds {
			nToks := len(e.Encoder.Decode(pred.TokenIDs))
			byTokenLen[nToks] = append(byTokenLen[nToks], pred)
		}

		for _, preds := range byTokenLen {
			sort.SliceStable(preds, func(i, j int) bool {
				return preds[i].Prob > preds[j].Prob
			})
		}

		var inTop1, inTopK int
		for nToks, preds := range byTokenLen {
			if nToks > len(labelTokens) {
				continue
			}
			labelBPs := e.Encoder.EncodeTokens(labelTokens[:nToks])
			metric := matchSeries(preds, labelBPs, e.TopN)

			if metric.InTopK == 1 && nToks > inTopK {
				inTopK = nToks
			}
			if metric.Accurate == 1 && nToks > inTop1 {
				inTop1 = nToks
			}
		}

		summary[CorrectTokensAhead] = append(summary[CorrectTokensAhead], Metric{Accurate: float64(inTop1), InTopK: float64(inTopK)})
	}

	e.evalSummary.append(summary)
	return nil
}

// Eval evaluates a file and updates the evaluator
func (e *Evaluator) Eval(sites PredictionSites) error {
	if e.evalSummary == nil {
		e.evalSummary = make(Summary)
	}

	if e.groupedTokenValueAdded == nil {
		e.groupedTokenValueAdded = make(map[int][]Metric)
	}

	err := e.metrics(sites)
	if err != nil {
		return errors.Wrapf(err, "error computing metrics for file %s", e.Filename)
	}

	if err := e.correctTokensAhead(sites, 8); err != nil {
		return errors.Wrapf(err, "error computing correct tokens ahead for file %s", e.Filename)
	}

	return nil
}

func (e *Evaluator) maybeMuddle(tokens []lexer.Token) []lexer.Token {
	if e.MinMuddle == 0 {
		return tokens
	}

	if len(tokens) < e.MinMuddle {
		return tokens
	}

	if len(tokens) > e.UnmuddledWindow {
		tokens = tokens[len(tokens)-e.UnmuddledWindow:]
	}

	var muddled []int
	for len(muddled) < e.Search.Window-e.UnmuddledWindow {
		muddled = append(muddled, rand.Intn(e.Predictor.GetHParams().VocabSize))
	}
	muddledToks := e.Encoder.Decode(muddled)

	return append(muddledToks, tokens...)
}
