package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/alexflint/go-arg"
	"github.com/gocarina/gocsv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type comparison struct {
	Func          string
	NumSeenAttrs  int
	NumValidAttrs []int
}

type results map[string]*summary

type summary struct {
	Count           int
	Valid           []int
	TotalSeenAttrs  int
	TotalValidAttrs []int
}

func max(nums []int) int {
	maxNum := nums[0]
	for _, num := range nums {
		if num > maxNum {
			maxNum = num
		}
	}
	return maxNum
}

func (re results) addComparison(comp comparison) {
	sym := comp.Func
	_, ok := re[sym]
	if !ok {
		re[sym] = &summary{
			Count:           0,
			Valid:           make([]int, len(comp.NumValidAttrs)),
			TotalSeenAttrs:  0,
			TotalValidAttrs: make([]int, len(comp.NumValidAttrs)),
		}
	}
	re[sym].Count++
	re[sym].TotalSeenAttrs += comp.NumSeenAttrs
	for i, n := range comp.NumValidAttrs {
		re[sym].TotalValidAttrs[i] += n
		if n == comp.NumSeenAttrs {
			re[sym].Valid[i]++
		}
	}
}

func validateAttrs(rm pythonresource.Manager, dist keytypes.Distribution, path pythonimports.DottedPath, attrs []string) int {
	var numValid int
	ps, err := rm.NewSymbol(dist, path)
	if err != nil {
		return 0
	}
	for _, attr := range attrs {
		_, err := rm.ChildSymbol(ps, attr)
		if err == nil {
			numValid++
		}
	}
	return numValid
}

// Example ...
type Example struct {
	Pkg   string
	Func  string
	Attrs []string
}

func compare(rm pythonresource.Manager, ex Example, model map[string][]pythoncode.EMType, k int) comparison {
	pred, ok := model[ex.Func]
	if !ok {
		return comparison{}
	}

	if len(pred) > k {
		pred = pred[:k]
	}

	var valids []int
	for _, p := range pred {
		valids = append(valids, validateAttrs(rm, p.Dist, p.Sym, ex.Attrs))
	}

	return comparison{
		Func:          ex.Func,
		NumSeenAttrs:  len(ex.Attrs),
		NumValidAttrs: valids,
	}
}

type report struct {
	TopKValidRatio  float64 `json:"topk_valid_ratio"`
	TopKAttrsRatio  float64 `json:"topk_attrs_ratio"`
	FirstValidRatio float64 `json:"first_valid_ratio"`
	FirstAttrsRatio float64 `json:"first_attrs_ratio"`
}

type record struct {
	Package             string  `csv:"package"`
	AllFuncs            int     `csv:"all_funcs"`
	HaveExamples        int     `csv:"have_examples"`
	TotalAttrs          int     `csv:"-"`
	FirstValidFuncs     int     `csv:"first_valid_funcs"`
	FirstValidFuncRatio float64 `csv:"first_valid_func_ratio"`
	FirstValidAttrs     int     `csv:"-"`
	FirstValidAttrRatio float64 `csv:"first_valid_attr_ratio"`
	TopK                int     `csv:"topK"`
	TopKValidFuncs      int     `csv:"topK_valid_funcs"`
	TopKValidFuncRatio  float64 `csv:"topK_valid_func_ratio"`
	TopKValidAttrs      int     `csv:"-"`
	TopKValidAttrRatio  float64 `csv:"topK_valid_attr_ratio"`
}

func main() {
	args := struct {
		ModelDir    string
		Packages    string
		ExampleDir  string
		OutDir      string
		TopK        int
		MinExamples int
	}{
		ExampleDir:  pythoncode.TypeInductionValidateData,
		TopK:        5,
		MinExamples: 15,
	}

	arg.MustParse(&args)

	pkgs, err := traindata.LoadPackageList(args.Packages)
	fail(err)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	var records []record

	for _, p := range pkgs {
		// Load models from the path
		modelFile := fileutil.Join(args.ModelDir, fmt.Sprintf("%s.json", p))
		modelf, err := fileutil.NewCachedReader(modelFile)
		if err != nil {
			fmt.Printf("No model found for %v, skipping...\n", p)
			continue
		}

		var model []pythoncode.EMReturnTypes
		fail(json.NewDecoder(modelf).Decode(&model))

		modelForValidate := make(map[string][]pythoncode.EMType)
		for _, rt := range model {
			modelForValidate[rt.Func.String()] = rt.ReturnTypes
		}

		// Load validation examples
		exampleFile := fileutil.Join(args.ExampleDir, fmt.Sprintf("%s.json", p))
		exf, err := fileutil.NewCachedReader(exampleFile)
		if err != nil {
			fmt.Printf("No validation examples for %v, skipping...\n", p)
			continue
		}

		var samples []Example
		err = json.NewDecoder(exf).Decode(&samples)
		fail(err)

		re := make(results)
		for _, s := range samples {
			comp := compare(rm, s, modelForValidate, args.TopK)
			if comp.Func == "" {
				continue
			}
			re.addComparison(comp)
		}

		rec := record{
			Package:      p,
			AllFuncs:     len(model),
			HaveExamples: len(re),
			TopK:         args.TopK,
		}

		reps := make(map[string]report, 0)
		for f, c := range re {
			reps[f] = report{
				TopKValidRatio:  float64(max(c.Valid)) / float64(c.Count),
				TopKAttrsRatio:  float64(max(c.TotalValidAttrs)) / float64(c.TotalSeenAttrs),
				FirstValidRatio: float64(c.Valid[0]) / float64(c.Count),
				FirstAttrsRatio: float64(c.TotalValidAttrs[0]) / float64(c.TotalSeenAttrs),
			}

			if max(c.Valid) == c.Count {
				rec.TopKValidFuncs++
			}

			if c.Count == c.Valid[0] {
				rec.FirstValidFuncs++
			}

			rec.FirstValidAttrs += c.TotalValidAttrs[0]
			rec.TopKValidAttrs += max(c.TotalValidAttrs)
			rec.TotalAttrs += c.TotalSeenAttrs
		}

		rec.FirstValidFuncRatio = float64(rec.FirstValidFuncs) / float64(rec.HaveExamples)
		rec.TopKValidFuncRatio = float64(rec.TopKValidFuncs) / float64(rec.HaveExamples)
		rec.FirstValidAttrRatio = float64(rec.FirstValidAttrs) / float64(rec.TotalAttrs)
		rec.TopKValidAttrRatio = float64(rec.TopKValidAttrs) / float64(rec.TotalAttrs)

		records = append(records, rec)

		if rec.HaveExamples < args.MinExamples {
			fmt.Printf("Only have %d examples, skipping...\n", rec.HaveExamples)
			continue
		}

		outFile := fileutil.Join(args.OutDir, fmt.Sprintf("%s.json", p))
		out, err := fileutil.NewBufferedWriter(outFile)
		fail(err)
		fail(json.NewEncoder(out).Encode(reps))
		out.Close()
	}

	overall := fileutil.Join(args.OutDir, "overall.csv")
	allout, err := fileutil.NewBufferedWriter(overall)
	fail(err)
	defer allout.Close()

	fail(gocsv.Marshal(&records, allout))
}
