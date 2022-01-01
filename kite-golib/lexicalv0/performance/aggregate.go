package performance

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func median(seq []int) float64 {
	if len(seq) == 0 {
		return 0
	}
	if len(seq)%2 == 0 {
		return (float64(seq[len(seq)/2-1]) + float64(seq[len(seq)/2])) / 2
	}
	return float64(seq[len(seq)/2])
}

type encLenStats struct {
	MedianPos float64
	MedianNeg float64
	MeanPos   float64
	MeanNeg   float64
	Mean      float64
	Median    float64
}

func (e *Evaluator) encLenStats() encLenStats {
	sort.Slice(e.encLen.Positive, func(i, j int) bool {
		return e.encLen.Positive[i] > e.encLen.Positive[j]
	})
	sort.Slice(e.encLen.Negative, func(i, j int) bool {
		return e.encLen.Negative[i] > e.encLen.Negative[j]
	})

	var posTotal int
	var negTotal int

	for _, l := range e.encLen.Positive {
		posTotal += l
	}

	for _, l := range e.encLen.Negative {
		negTotal += l
	}

	all := append([]int{}, e.encLen.Positive...)
	all = append(all, e.encLen.Negative...)

	return encLenStats{
		MedianPos: median(e.encLen.Positive),
		MedianNeg: median(e.encLen.Negative),
		MeanPos:   float64(posTotal) / float64(len(e.encLen.Positive)),
		MeanNeg:   float64(negTotal) / float64(len(e.encLen.Negative)),
		Mean:      float64(posTotal+negTotal) / (float64(len(e.encLen.Positive)) + float64(len(e.encLen.Negative))),
		Median:    median(all),
	}
}

type aggregated struct {
	num       int
	accuracy  float64
	accuracyK float64
	numErrs   int
}

func aggregateMetrics(metrics []Metric) aggregated {
	var sumAcc float64
	var sumTopK float64
	var errs int
	for _, m := range metrics {
		sumAcc += m.Accurate
		sumTopK += m.InTopK
		if m.Err != nil {
			errs++
		}
	}
	return aggregated{
		num:       len(metrics),
		accuracy:  sumAcc / float64(len(metrics)-errs),
		accuracyK: sumTopK / float64(len(metrics)-errs),
		numErrs:   errs,
	}
}

func format(category Measurement, agg aggregated) string {
	parts := []string{
		fmt.Sprintf("%s_num\t%d", category, agg.num),
		fmt.Sprintf("%s_num_errs\t%d", category, agg.numErrs),
		fmt.Sprintf("%s_accuracy\t%.5f", category, agg.accuracy),
		fmt.Sprintf("%s_topk_accuracy\t%.5f", category, agg.accuracyK),
	}
	return strings.Join(parts, "\n") + "\n"
}

func percentiles(seq []float64, chunks int) string {
	sort.Float64s(seq)
	var pctls []string
	total := len(seq)
	for i := 1; i < chunks; i++ {
		if len(seq) == 0 {
			pctls = append(pctls, fmt.Sprintf("%f", math.NaN()))
		} else {
			pctls = append(pctls, fmt.Sprintf("%.2f", seq[i*total/chunks]))
		}
	}
	return strings.Join(pctls, ",")
}

// AggregateAndWrite the results of evaluator and save to file.
func AggregateAndWrite(evaluators []Evaluator, f *os.File, detailed bool) error {
	sep := "\t"
	writeRows := func(rows [][]string) {
		for _, row := range rows {
			fmt.Fprintf(f, "%s\n", strings.Join(row, sep))
		}
	}

	keyForAll := ".all"
	byExt := combineEvaluators(evaluators, keyForAll)

	var exts []string
	for ext := range byExt {
		exts = append(exts, ext)
	}
	sort.Strings(exts)

	fmt.Fprintf(f, "Series length\t%d\n", evaluators[0].SeriesLength)
	// extensions across the top, metrics on the left hand side
	fmt.Fprintf(f, "Ext%s%s\n", sep, strings.Join(exts, sep))

	e := byExt[keyForAll]

	// Top1 and TopK for all metrics except perplexity
	var mKeys []Measurement
	for mea := range e.Measurements {
		if e.Measurements[mea] == 0 {
			continue
		}
		mKeys = append(mKeys, mea)
	}

	sort.Slice(mKeys, func(i, j int) bool {
		return string(mKeys[i]) < string(mKeys[j])
	})

	for _, mea := range mKeys {
		rows := [][]string{
			{fmt.Sprintf("%s_num", mea)},
			{fmt.Sprintf("%s_num_errs", mea)},
			{fmt.Sprintf("%s_accuracy", mea)},
			{fmt.Sprintf("%s_topk_accuracy", mea)},
		}
		for _, ext := range exts {
			e := byExt[ext]
			agg := aggregateMetrics(e.evalSummary[mea])
			rows[0] = append(rows[0], fmt.Sprintf("%v", agg.num))
			rows[1] = append(rows[1], fmt.Sprintf("%v", agg.numErrs))
			rows[2] = append(rows[2], fmt.Sprintf("%v", agg.accuracy))
			rows[3] = append(rows[3], fmt.Sprintf("%v", agg.accuracyK))
		}
		writeRows(rows)
	}

	// Encoding lengths
	if e.Measurements[Word] > 0 || e.Measurements[String] > 0 {
		rows := [][]string{
			{"median_num_BP_pos_word"},
			{"median_num_BP_neg_word"},
			{"mean_num_BP_pos_word"},
			{"mean_num_BP_neg_word"},
			{"mean_num_BP_word"},
			{"median_num_BP_word"},
		}
		for _, ext := range exts {
			e := byExt[ext]
			agg := e.encLenStats()
			rows[0] = append(rows[0], fmt.Sprintf("%v", agg.MedianPos))
			rows[1] = append(rows[1], fmt.Sprintf("%v", agg.MedianNeg))
			rows[2] = append(rows[2], fmt.Sprintf("%v", agg.MeanPos))
			rows[3] = append(rows[3], fmt.Sprintf("%v", agg.MeanNeg))
			rows[4] = append(rows[4], fmt.Sprintf("%v", agg.Mean))
			rows[5] = append(rows[5], fmt.Sprintf("%v", agg.Median))
		}

		writeRows(rows)
	}

	if !detailed {
		return nil
	}

	// TVA percentiles
	if e.Measurements[TokenValueAdded] > 0 {
		rows := [][]string{
			{"token_value_added_acc_percentile"},
			{"token_value_added_topk_percentile"},
		}
		for _, ext := range exts {
			e := byExt[ext]
			var taccs []float64
			var ttopks []float64
			for _, m := range e.evalSummary[TokenValueAdded] {
				taccs = append(taccs, m.Accurate)
				ttopks = append(ttopks, m.InTopK)
			}
			rows[0] = append(rows[0], percentiles(taccs, 10))
			rows[1] = append(rows[1], percentiles(ttopks, 10))
		}
		writeRows(rows)

		var tvaGroups []int
		for k := range e.groupedTokenValueAdded {
			tvaGroups = append(tvaGroups, k)
		}
		sort.Ints(tvaGroups)

		rows = nil
		for _, key := range tvaGroups {
			rows = append(rows, []string{fmt.Sprintf("token_value_added_%d_idents_num", key)})
			rows = append(rows, []string{fmt.Sprintf("token_value_added_%d_idents_acc", key)})
			rows = append(rows, []string{fmt.Sprintf("token_value_added_%d_idents_topk", key)})
		}
		for i, key := range tvaGroups {
			start := i * 3
			for _, ext := range exts {
				e := byExt[ext]
				agg := aggregateMetrics(e.groupedTokenValueAdded[key])
				rows[start+0] = append(rows[start+0], fmt.Sprintf("%v", agg.num))
				rows[start+1] = append(rows[start+1], fmt.Sprintf("%v", agg.accuracy))
				rows[start+2] = append(rows[start+2], fmt.Sprintf("%v", agg.accuracyK))
			}
		}
		writeRows(rows)
	}

	// CVA percentiles
	if e.Measurements[CharValueAdded] > 0 {
		rows := [][]string{
			{"char_value_added_acc_percentile"},
			{"char_value_added_topk_percentile"},
		}
		for _, ext := range exts {
			e := byExt[ext]
			var taccs []float64
			var ttopks []float64
			for _, m := range e.evalSummary[CharValueAdded] {
				taccs = append(taccs, m.Accurate)
				ttopks = append(ttopks, m.InTopK)
			}
			rows[0] = append(rows[0], percentiles(taccs, 10))
			rows[1] = append(rows[1], percentiles(ttopks, 10))
		}
		writeRows(rows)
	}

	// Correct tokens ahead percentiles
	if e.Measurements[CorrectTokensAhead] > 0 {
		rows := [][]string{
			{"correct_tokens_ahead_acc_percentile"},
			{"correct_tokens_ahead_topk_percentile"},
		}
		for _, ext := range exts {
			e := byExt[ext]
			var taccs []float64
			var ttopks []float64
			for _, m := range e.evalSummary[CorrectTokensAhead] {
				taccs = append(taccs, m.Accurate)
				ttopks = append(ttopks, m.InTopK)
			}
			rows[0] = append(rows[0], percentiles(taccs, 10))
			rows[1] = append(rows[1], percentiles(ttopks, 10))
		}
		writeRows(rows)
	}
	return nil
}

// combineEvaluators combines several evaluator to one (make sure they all have the same measurements),
// and also includes a breakdown of evaluator metrics by extension
func combineEvaluators(evaluators []Evaluator, keyForAll string) map[string]Evaluator {
	newEvaluator := func() Evaluator {
		return Evaluator{
			evalSummary:            make(Summary),
			Measurements:           evaluators[0].Measurements,
			groupedTokenValueAdded: make(map[int][]Metric),
			report:                 "",
		}
	}

	agg := func(base, e Evaluator) Evaluator {
		base.evalSummary.append(e.evalSummary)
		base.totalEntropy += e.totalEntropy
		base.totalWords += e.totalWords
		base.encLen.Positive = append(base.encLen.Positive, e.encLen.Positive...)
		base.encLen.Negative = append(base.encLen.Negative, e.encLen.Negative...)
		for key := range e.groupedTokenValueAdded {
			base.groupedTokenValueAdded[key] = append(base.groupedTokenValueAdded[key], e.groupedTokenValueAdded[key]...)
		}
		base.report += e.report
		return base
	}

	byExt := make(map[string]Evaluator)
	for _, e := range evaluators {
		for _, ext := range []string{keyForAll, filepath.Ext(e.Filename)} {
			if _, ok := byExt[ext]; !ok {
				byExt[ext] = newEvaluator()
			}
			byExt[ext] = agg(byExt[ext], e)
		}
	}

	if len(byExt) == 2 {
		// only one extension
		return map[string]Evaluator{
			keyForAll: byExt[keyForAll],
		}
	}

	return byExt
}
