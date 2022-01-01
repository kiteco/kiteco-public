//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	arg "github.com/alexflint/go-arg"
	"github.com/codegangsta/negroni"
	"github.com/go-errors/errors"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	parseOpts = pythonparser.Options{
		Approximate: true,
		ErrorMode:   pythonparser.Recover,
	}
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func selectedBuffer(src string) (data.SelectedBuffer, error) {
	var sb data.SelectedBuffer
	switch parts := strings.Split(src, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(src).Select(data.Cursor(len(src)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
		return sb, errors.Errorf("Too many '$' in the input")
	}
	return sb, nil
}

type results struct {
	inputs pythonproviders.Inputs
	//completion []string
	completion *returnedComp
}

type returnedComp struct {
	Completion     string
	Replace        int
	MetaCompletion pythonproviders.MetaCompletion
	SkipCompletion bool
	PropagatedSkip bool
	Children       []*returnedComp
	Tooltip        string
	GGNNProb       float64
	CallProb       float64
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.ico")
}

func main() {
	fail(datadeps.Enable())
	args := struct {
		Port  string
		Input string
	}{
		Port:  ":3037",
		Input: "snippet.py",
	}

	arg.MustParse(&args)
	src, err := ioutil.ReadFile(args.Input)
	fail(err)
	log.Printf("successful reading %v", args.Input)
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}
	app, err := newApp(rm, string(src))
	fail(err)
	r := mux.NewRouter()
	r.HandleFunc("/", app.handleRequest)
	r.HandleFunc("/favicon.ico", faviconHandler)

	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)

	log.Printf("listening on http://localhost%s\n", args.Port)
	fail(http.ListenAndServe(args.Port, neg))
}

func sortCompletions(completions *returnedComp, ordering string) {
	switch ordering {
	case "callprob":
		sortByCallProb(completions)
	case "combined-prob":
		sortByCombinedProb(completions)
	case "alphabetical":
		sortAlphabeticalNested(completions)
	case "provider":
		sortFromProvider(completions)
	default:
		fmt.Println("WARNING: Unknown ordering : ", ordering)
		sortAlphabeticalNested(completions)
	}
}

func sortFromProvider(comp *returnedComp) {
	flattenCompletions(comp, false)
	sort.Slice(comp.Children, func(i, j int) bool {
		c1, c2 := comp.Children[i], comp.Children[j]
		return higherPriority(c1.MetaCompletion, c2.MetaCompletion)
	})
}

func sortByCombinedProb(comp *returnedComp) {
	flattenCompletions(comp, false)
	sort.Slice(comp.Children, func(i, j int) bool {
		c1, c2 := comp.Children[i], comp.Children[j]
		return c1.CallProb*c1.GGNNProb > c2.CallProb*c2.GGNNProb
	})
}

func sortByCallProb(comp *returnedComp) {
	flattenCompletions(comp, false)
	sort.Slice(comp.Children, func(i, j int) bool {
		c1, c2 := comp.Children[i], comp.Children[j]
		return c1.CallProb > c2.CallProb
	})
}

func flattenCompletions(completions *returnedComp, includeItself bool) []*returnedComp {
	var result []*returnedComp
	for _, c := range completions.Children {
		result = append(result, flattenCompletions(c, true)...)
	}
	if includeItself {
		completions.Children = nil
		result = append(result, completions)
	} else {
		completions.Children = result
	}
	return result
}

func sortAlphabeticalNested(completions *returnedComp) {
	if len(completions.Children) == 0 {
		completions.Completion = strings.Replace(completions.Completion, ",)", ")", -1)
		return
	}
	for _, c := range completions.Children {
		c.Completion = strings.Replace(c.Completion, ",)", ")", -1)
		sortAlphabeticalNested(c)
	}
	sort.Slice(completions.Children, func(i, j int) bool {
		return completions.Children[i].Completion < completions.Children[j].Completion
	})
	return
}

func higherPriority(comp1, comp2 pythonproviders.MetaCompletion) bool {
	if comparator, ok := comp1.MixingMeta.Provider.Provider.(pythonproviders.CompletionComparator); ok {
		// If completion comes from the same provider and it is implementing CompletionComparator
		// we used that to sort them
		return comparator.CompareCompletions(comp1, comp2)
	}
	return comp1.Snippet.Text < comp2.Snippet.Text
}

func (a app) getCallModelCompletions(src, ordering string, filtering bool) (*returnedComp, int) {
	global := pythonproviders.Global{
		FilePath:        "/src.py",
		ResourceManager: a.rm,
		Models:          a.models,
		Product:         licensing.Pro,
	}

	//prepare selected buffer
	sb, err := selectedBuffer(string(src))
	if err != nil {
		return &returnedComp{
			Completion: fmt.Sprint(err),
		}, 0
	}
	inputs, err := pythonproviders.NewInputs(kitectx.Background(), global, sb, false, true)
	provider := pythonproviders.CallModel{}
	root := &returnedComp{
		MetaCompletion: pythonproviders.MetaCompletion{
			Completion: data.Completion{
				Snippet: data.Snippet{},
				Replace: data.Cursor(0),
			},
		},
	}
	err = provider.Provide(kitectx.Background(), global, inputs, func(ctx kitectx.Context, sb data.SelectedBuffer, mc pythonproviders.MetaCompletion) {
		root.Children = append(root.Children, &returnedComp{
			Completion:     mc.Completion.Snippet.Text,
			Replace:        0,
			MetaCompletion: mc,
			SkipCompletion: false,
			PropagatedSkip: false,
			Children:       nil,
			Tooltip:        getTooltip(mc),
			GGNNProb:       mc.Score,
			CallProb:       mc.CallModelMeta.CallProb,
		})
	})
	if err != nil {
		fmt.Println(err)
	}

	sortCompletions(root, ordering)

	return root, 0

}

func (a app) getCompletions(src, ordering string, filtering bool) (*returnedComp, int) {
	log.Printf("source is:\n%v", string(src))
	global := pythonproviders.Global{
		FilePath:        "/src.py",
		ResourceManager: a.rm,
		Models:          a.models,
		Product:         licensing.Pro,
	}

	//prepare selected buffer
	sb, err := selectedBuffer(string(src))
	if err != nil {
		return &returnedComp{
			Completion: fmt.Sprint(err),
		}, 0
	}
	inputs, err := pythonproviders.NewInputs(kitectx.Background(), global, sb, false, true)
	provider := pythonproviders.GGNNModel{ForceDisableFiltering: !filtering}
	root := &returnedComp{
		MetaCompletion: pythonproviders.MetaCompletion{
			Completion: data.Completion{
				Snippet: data.Snippet{},
				Replace: data.Cursor(0),
			},
		},
	}
	workingQueue := []results{results{inputs: inputs, completion: root}}
	seen := make(map[string]bool)
	for len(workingQueue) > 0 {
		// pop
		inputs, baseComp := workingQueue[0].inputs, workingQueue[0].completion
		workingQueue = workingQueue[1:]
		err = provider.Provide(kitectx.Background(), global, inputs, func(ctx kitectx.Context, sb data.SelectedBuffer, mc pythonproviders.MetaCompletion) {
			child := composeCompletion(baseComp, mc)

			if !seen[child.Completion] || mc.Snippet.Text == "" {
				baseComp.Children = append(baseComp.Children, child)
				seen[child.Completion] = true
			}
			if p := mc.GGNNMeta.Predictor; p != nil {
				var nextBuffer data.SelectedBuffer
				if mc.GGNNMeta.SpeculationPlaceholderPresent {
					placeholders := mc.Snippet.Placeholders()
					nextBuffer = inputs.Select(mc.Replace).Replace(mc.Snippet.Text).Select(data.Cursor(inputs.Begin + placeholders[len(placeholders)-1].Begin))
				} else {
					nextBuffer = inputs.Select(mc.Replace).ReplaceWithCursor(mc.Snippet.Text)
				}
				nextInputs, err := pythonproviders.NewInputs(kitectx.Background(), global, nextBuffer, false, true)
				if err != nil {
					log.Println(err)
					return
				}
				nextInputs.GGNNPredictor = p
				workingQueue = append(workingQueue, results{
					inputs:     nextInputs,
					completion: child,
				})
			}
		})
		if err != nil {
			log.Println(err)
			return &returnedComp{
				Completion: fmt.Sprintf("Error while getting completions : %s", err),
			}, 0
		}
	}
	var skippedCompletions int
	if !filtering {
		skippedCompletions = propagateSkipCompletions(root, false)
	}
	sortCompletions(root, ordering)
	return root, skippedCompletions
}

func composeCompletion(baseComp *returnedComp, newCompletion pythonproviders.MetaCompletion) *returnedComp {
	result := returnedComp{}
	newComp := newCompletion.Completion
	if baseComp.Completion != "" {
		newComp = newCompletion.MustAfter(baseComp.MetaCompletion.Completion)
	} else {

	}
	result.Completion = newComp.Snippet.Text
	newCompletion.Completion = newComp
	result.MetaCompletion = newCompletion

	var skipCall bool
	var ggnnProb float64
	if newCompletion.GGNNMeta.Call != nil {
		skipCall = newCompletion.GGNNMeta.Call.SkipCall
		ggnnProb = float64(newCompletion.GGNNMeta.Call.Prob)
	}
	result.SkipCompletion = skipCall
	result.Tooltip = getTooltip(newCompletion)
	result.CallProb = newCompletion.Score
	result.GGNNProb = ggnnProb
	result.MetaCompletion = newCompletion

	return &result
}

func propagateSkipCompletions(comp *returnedComp, forceSkip bool) int {
	var result int
	if forceSkip {
		if !comp.SkipCompletion {
			result++
			comp.PropagatedSkip = true
		}
		comp.SkipCompletion = true
	} else if comp.SkipCompletion {
		forceSkip = true
	}
	for _, c := range comp.Children {
		result += propagateSkipCompletions(c, forceSkip)
	}
	return result
}

func getTooltip(mc pythonproviders.MetaCompletion) string {
	var result string

	if mc.GGNNMeta != nil && mc.GGNNMeta.Call != nil {
		result += fmt.Sprintf("Call Prob : %v EG Prob : %v\n", mc.GGNNMeta.Call.CallProb, mc.GGNNMeta.Call.Prob)
		result += "Features: \n"
		for _, f := range mc.GGNNMeta.Call.MetaData.FilteringFeatures {
			result += fmt.Sprintln(f.String())
		}
		result += "\nModel Weigths : \n"
		for _, w := range mc.GGNNMeta.Call.MetaData.ModelWeight {
			result += fmt.Sprintln(w.String())
		}
	}
	if mc.CallModelMeta != nil && mc.CallModelMeta.Call != nil {
		result += fmt.Sprintf("Call Prob : %v CallModel Prob : %v\n", mc.CallModelMeta.CallProb, mc.CallModelMeta.Call.Prob)
		result += "Features: \n"
		for _, f := range mc.CallModelMeta.Call.MetaData.FilteringFeatures {
			result += fmt.Sprintln(f.String())
		}
		result += "\nModel Weigths : \n"
		for _, w := range mc.CallModelMeta.Call.MetaData.ModelWeight {
			result += fmt.Sprintln(w.String())
		}
	}
	return result
}
