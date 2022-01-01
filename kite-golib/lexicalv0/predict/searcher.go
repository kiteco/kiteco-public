package predict

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

// clients should not modify candidates, clients should not hold references to returned logits
type nextLogitsFn func(ctx kitectx.Context, step int, candidates [][]int64) ([][]float32, error)

type metricsFn func(Predicted) PredictedMetrics

type searcher struct {
	config      SearchConfig
	nextLogits  nextLogitsFn
	predictChan chan Predicted
	enc         *lexicalv0.FileEncoder
	prefix      string
	rand        *rand.Rand
	strict      bool
	metricsFn   metricsFn

	includeMetaInfo bool
}

type candidate struct {
	TokenIDs []int64
	Prob     float32
}

type candidates []candidate

func (cs candidates) batch() [][]int64 {
	b := make([][]int64, 0, len(cs))
	for _, c := range cs {
		b = append(b, c.TokenIDs)
	}
	return b
}

func (cs candidates) toPredicted(prefix string, encoder *lexicalv0.FileEncoder, metricsFn metricsFn) []Predicted {
	if len(cs) == 0 {
		return nil
	}

	ps := make([]Predicted, 0, len(cs))
	for _, c := range cs {
		p := newPredicted(c.TokenIDs, c.Prob, prefix, encoder, false)
		if metricsFn != nil {
			p.Metrics = metricsFn(p)
		}

		ps = append(ps, p)
	}

	sort.Slice(ps, func(i, j int) bool {
		return ps[i].Prob > ps[j].Prob
	})

	return ps
}

func (s searcher) Search(ctx kitectx.Context) (Predictions, error) {
	if s.predictChan != nil {
		defer close(s.predictChan)
	}
	// check abort _after_ we defer close the channel to ensure that
	// the predictChan is not leaked if we are aborted.
	ctx.CheckAbort()

	var cs candidates
	var meta PredictionsMeta
	for i := 0; i < s.config.Depth; i++ {
		var err error
		var exp Expansion
		cs, exp, err = s.step(ctx, i, cs)
		if err != nil {
			return Predictions{}, err
		}

		if s.includeMetaInfo {
			meta.Expansions = append(meta.Expansions, exp)
		}

		if len(cs) == 0 {
			// no more candidates, all of them must have been filtered,
			// so we are done
			break
		}

		if s.predictChan != nil {
			for _, pred := range cs.toPredicted(s.prefix, s.enc, s.metricsFn) {
				s.predictChan <- pred
			}
		}
	}

	return Predictions{
		Preds: cs.toPredicted(s.prefix, s.enc, s.metricsFn),
		Meta:  meta,
	}, nil
}

func (s searcher) step(ctx kitectx.Context, stepNum int, cands candidates) (candidates, Expansion, error) {
	ctx.CheckAbort()

	// shape [current width, vocab]
	logits, err := s.nextLogits(ctx, stepNum, cands.batch())
	if err != nil {
		return nil, Expansion{}, errors.Wrapf(err, "error getting logits for step %d", stepNum)
	}

	// shape [current width, vocab]
	batchExts := newBatchExts(logits)
	if s.config.UseTemperatureScaling {
		batchExts.temperatureScale(s.enc.IsLexical, s.config.LexicalTemperature, s.config.IdentTemperature)
		batchExts.softmax()
	}

	var exp Expansion
	if s.includeMetaInfo {
		exp.AtDepth = stepNum
		exp.Input = cands.toPredicted(s.prefix, s.enc, s.metricsFn)
		exp.RawPredictions = batchExts.toPredicted(s.prefix, s.enc, s.metricsFn)
	}

	// prefix filtering
	if stepNum == 0 && s.prefix != "" {
		batchExts = s.filterByPrefix(ctx, batchExts)
		if len(batchExts) == 0 {
			return nil, exp, nil
		}
		batchExts.normalize(s.config.PrefixRegularization)
	}

	// Two step sampling selection logic from https://arxiv.org/pdf/1701.03185.pdf
	// 1) sample expansions to the existing hyptheses (beams), shape [current width, min(beam width, num candidates surviving minp, topk)]
	// 2) construct all possible new hypoptheses, shape [current width*min(beam width, num candidates surviving minp, topk)]
	// 3) sample from set of all possible new hypotheses, shape [min(beam width, num candidates surviving minp, topk)]
	cands, batchExts = s.selectExtensions(ctx, cands, batchExts)
	if len(batchExts) == 0 {
		// need to check batchExts above since cands can be len 0 for first iter
		return nil, exp, nil
	}
	if s.includeMetaInfo {
		exp.SelectedPredictions = batchExts.toPredicted(s.prefix, s.enc, s.metricsFn)
	}

	newCands := s.allExpansions(ctx, cands, batchExts)
	if len(newCands) == 0 {
		return nil, exp, nil
	}

	// do not double sample for the first step since it does nothing
	if stepNum == 0 {
		if s.includeMetaInfo {
			exp.BeamPredictions = newCands.toPredicted(s.prefix, s.enc, s.metricsFn)
		}

		return newCands, exp, nil
	}

	newCands = s.selectCandidates(ctx, newCands)
	if len(newCands) == 0 {
		return nil, exp, nil
	}
	if s.includeMetaInfo {
		exp.BeamPredictions = newCands.toPredicted(s.prefix, s.enc, s.metricsFn)
	}

	return newCands, exp, nil
}

func (s searcher) selectExtensions(ctx kitectx.Context, base candidates, bExts batchExts) (candidates, batchExts) {
	ctx.CheckAbort()

	return s.filterExts(ctx, base, bExts, func(exts candidates) candidates {
		return s.selectCandidates(ctx, exts)
	})
}

func (s searcher) selectCandidates(ctx kitectx.Context, cs candidates) candidates {
	ctx.CheckAbort()

	selected := make(candidates, 0, s.config.BeamWidth)

	// minp
	for _, c := range cs {
		if c.Prob < s.config.MinP {
			continue
		}
		selected = append(selected, c)
	}

	// topk
	if len(selected) > s.config.TopK {
		sort.SliceStable(selected, func(i, j int) bool {
			return selected[i].Prob > selected[j].Prob
		})
		selected = selected[:s.config.TopK]
	}

	// sample without replacement
	return s.sample(ctx, selected, s.config.BeamWidth)
}

// NOTE: cs may be modified when calling sample
func (s searcher) sample(ctx kitectx.Context, cs candidates, numSamples int) candidates {
	ctx.CheckAbort()

	if len(cs) <= numSamples {
		return cs
	}

	sampled := make(candidates, 0, numSamples)
	for i := 0; i < numSamples; i++ {
		idx := s.drawSample(cs)
		if idx == -1 {
			// something went terribly wrong in sampling
			continue
		}

		sampled = append(sampled, cs[idx])
		cs[idx].Prob = 0
	}

	return sampled
}

func (s searcher) drawSample(cs candidates) int {
	cdf := make([]float32, 1, len(cs)+1)
	for i, c := range cs {
		cdf = append(cdf, c.Prob+cdf[i])
	}

	prob := s.rand.Float32() * cdf[len(cdf)-1]
	for i := 0; i < len(cdf)-1; i++ {
		if prob >= cdf[i] && prob < cdf[i+1] {
			return i
		}
	}

	if s.strict {
		panic(fmt.Sprintf("unable to sample, %v, %v, %v\n", prob, cs, cdf))
	}

	return -1
}

func (s searcher) allExpansions(ctx kitectx.Context, base candidates, batchExts batchExts) candidates {
	ctx.CheckAbort()

	if !s.baseAndExtsOK(base, batchExts, "all expansions") {
		return nil
	}

	if len(base) == 0 {
		if len(batchExts) != 1 {
			if s.strict {
				panic(fmt.Sprintf("expected batchExts to have len 1 when base is empty, but got %d", len(batchExts)))
			}
			return nil
		}

		// this only happens on the first iteration so we can safely just return
		return batchExts[0]
	}

	var exps candidates
	for i, exts := range batchExts {
		if i == 0 {
			exps = make(candidates, 0, len(base)*len(exts))
		}

		baseProb := base[i].Prob
		baseCand := base[i].TokenIDs
		for _, ext := range exts {
			newProb := baseProb * ext.Prob
			if newProb < 1e-8 {
				// prevent numerical underflow
				continue
			}
			exps = append(exps, candidate{
				TokenIDs: copyAndAppend(baseCand, ext.TokenIDs),
				Prob:     newProb,
			})
		}
	}

	return exps
}

// no need to track base candidates because this is only called on step 0
// when base candidates is empty
func (s searcher) filterByPrefix(ctx kitectx.Context, bExts batchExts) batchExts {
	ctx.CheckAbort()

	validIDs := idsMatchingPrefix(s.enc, s.prefix)
	_, bExts = s.filterExts(ctx, nil, bExts, func(exts candidates) candidates {
		var keep candidates
		for _, ext := range exts {
			if validIDs[int(ext.TokenIDs[0])] {
				keep = append(keep, ext)
			}
		}
		return keep
	})
	return bExts
}

func (s searcher) filterExts(ctx kitectx.Context, base candidates, bExts batchExts, filter func(candidates) candidates) (candidates, batchExts) {
	ctx.CheckAbort()
	if !s.baseAndExtsOK(base, bExts, "in filter exts") {
		return nil, nil
	}

	// make sure we keep ordering of base candidates consistent
	keepBase := make(candidates, 0, len(bExts))
	keepExts := make(batchExts, 0, len(bExts))

	for i, exts := range bExts {
		filtered := filter(exts)
		if len(filtered) == 0 {
			continue
		}

		keepExts = append(keepExts, filtered)
		if len(base) > 0 {
			keepBase = append(keepBase, base[i])
		}
	}

	return keepBase, keepExts
}

func (s searcher) baseAndExtsOK(base candidates, bExts batchExts, msg string) bool {
	bad := len(base) > 0 && len(base) != len(bExts)
	if bad && s.strict {
		panic(fmt.Sprintf("%s: batch sizes for base (%d) and extensions (%d) not equal", msg, len(base), len(bExts)))
	}
	return !bad
}

//
// --
//

type batchExts []candidates

func newBatchExts(batchScores [][]float32) batchExts {
	batchExts := make(batchExts, 0, len(batchScores))
	for _, scores := range batchScores {
		exts := make(candidates, 0, len(scores))
		for j, score := range scores {
			exts = append(exts, candidate{
				TokenIDs: []int64{int64(j)},
				Prob:     score,
			})
		}
		batchExts = append(batchExts, exts)
	}

	return batchExts
}

func (b batchExts) normalize(regularizer float32) {
	for _, exts := range b {
		sum := regularizer
		for _, ext := range exts {
			sum += ext.Prob
		}

		if sum > 1 {
			continue
		}

		invSum := 1 / sum
		for j := range exts {
			exts[j].Prob *= invSum
		}
	}
}

func (b batchExts) softmax() {
	for _, logits := range b {
		var max float32
		for _, logit := range logits {
			if logit.Prob > max {
				max = logit.Prob
			}
		}

		var sum float32
		for v, logit := range logits {
			prob := float32(math.Exp(float64(logit.Prob - max)))
			sum += prob
			logits[v].Prob = prob
		}

		invSum := 1. / sum
		for v := range logits {
			logits[v].Prob *= invSum
		}
	}
}

func (b batchExts) temperatureScale(isLexical func(int) bool, lexTemp, identTemp float32) {
	for _, cands := range b {
		for i, cand := range cands {
			if isLexical(int(cand.TokenIDs[0])) {
				cands[i].Prob /= lexTemp
			} else {
				cands[i].Prob /= identTemp
			}
		}
	}
}

func (b batchExts) toPredicted(prefix string, encoder *lexicalv0.FileEncoder, metricsFn metricsFn) [][]Predicted {
	batch := make([][]Predicted, 0, len(b))
	for _, exts := range b {
		batch = append(batch, exts.toPredicted(prefix, encoder, metricsFn))
	}
	return batch
}
