package main

import (
	"encoding/json"
	"log"
	"math"
	"sort"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

const (
	attrSmoothing       = .1
	numTypesPerFunction = 100
	defaultUsages       = "s3://kite-emr/users/tarak/python-code-examples/2015-10-21_13-13-06-PM/merge_group_obj_usages/output/part-00000"
)

// represents a python class together with a probability distribution over its attributes
type typeModel struct {
	id        int64              // ID in import graph
	name      string             // canonical name
	pkg       string             // top-level package that this type belongs to
	max       float64            // maximum number of times any one attribute was accessed
	instances []*variable        // list of variables with non-zero probability of being this type
	attrs     map[string]float64 //counts the number of times each attribute was accessed on this type
}

func newTypeModel(id int64, canonicalName string) *typeModel {
	return &typeModel{
		id:    id,
		name:  canonicalName,
		pkg:   root(canonicalName),
		attrs: make(map[string]float64),
	}
}

// represents a python function together with a probability distribution over possible return types
type funcModel struct {
	id          int64                  // ID in import graph
	name        string                 // canonical name
	pkg         string                 // top-level package that this function belongs to
	usages      []*variable            // list of variables that were constructed by this function
	returnTypes map[*typeModel]float64 // probability distribution over return types for this function
}

func newFuncModel(id int64, canonicalName string) *funcModel {
	return &funcModel{
		id:          id,
		name:        canonicalName,
		pkg:         root(canonicalName),
		returnTypes: make(map[*typeModel]float64),
	}
}

// represents a single variable that was returned from a function and then had
// certain attributes accessed on it
type variable struct {
	function     *funcModel             // function that this variable was returned from
	attrs        []string               // attributes accessed on this variable
	types        map[*typeModel]float64 // distribution over types
	observedType *typeModel             // true if this variable is from dynamic analysis
}

func newVariable(function *funcModel) *variable {
	return &variable{
		function: function,
		types:    make(map[*typeModel]float64),
	}
}

type pair struct {
	key   int64
	value float64
}

type byValue []pair

func (xs byValue) Len() int           { return len(xs) }
func (xs byValue) Less(i, j int) bool { return xs[i].value < xs[j].value }
func (xs byValue) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }

type byProbability []typeinduction.Element

func (xs byProbability) Len() int           { return len(xs) }
func (xs byProbability) Less(i, j int) bool { return xs[i].LogProbability < xs[j].LogProbability }
func (xs byProbability) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }

// get the topN items sorted by value
func topN(xs map[int64]float64, n int) []int64 {
	var pairs []pair
	for k, v := range xs {
		pairs = append(pairs, pair{k, v})
	}
	sort.Sort(sort.Reverse(byValue(pairs)))
	if len(pairs) > n {
		pairs = pairs[:n]
	}
	var out []int64
	for _, p := range pairs {
		out = append(out, p.key)
	}
	return out
}

func root(dotted string) string {
	if pos := strings.Index(dotted, "."); pos != -1 {
		return dotted[:pos]
	}
	return dotted
}

func contains(list []string, item string) bool {
	for _, x := range list {
		if x == item {
			return true
		}
	}
	return false
}

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func assertFinite(x float64, fmt string, args ...interface{}) {
	if !isFinite(x) {
		log.Fatalf(fmt, args...)
	}
}

func main() {
	var args struct {
		ImportGraph   string
		ImportDeps    string `arg:"required"`
		StaticUsages  string
		DynamicUsages string
		Funcs         string `arg:"required"`
		Types         string `arg:"required"`
		StaticLimit   int
		Steps         int
	}
	args.ImportGraph = pythonimports.DefaultImportGraph
	args.StaticUsages = defaultUsages
	args.Steps = 1
	arg.MustParse(&args)

	// Open the usages
	rr, err := fileutil.NewCachedReader(args.StaticUsages)
	if err != nil {
		log.Fatal(err)
	}
	defer rr.Close()

	// Open the output files early so that we fail fast if they can't be opened
	funcEnc, err := serialization.NewEncoder(args.Funcs)
	if err != nil {
		log.Fatalf("error leading %s: %v", args.Funcs, err)
	}
	defer funcEnc.Close()

	typeEnc, err := serialization.NewEncoder(args.Types)
	if err != nil {
		log.Fatal(err)
	}
	defer typeEnc.Close()

	// Load the dependency graph
	deps, err := pythonimports.LoadDependencies(args.ImportDeps)
	if err != nil {
		log.Fatal(err)
	}

	// Load the import graph from S3
	log.Println("Loading import graph...")
	graph, err := pythonimports.NewGraph(args.ImportGraph)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize maps
	types := make(map[int64]*typeModel)
	funcs := make(map[int64]*funcModel)
	var unobservedVars, observedVars []*variable // github and dynamic-analysis variables, respectively

	// Make a list of all types and build an index from attributes to types
	log.Println("Enumerating types from import graph...")
	typesByAttribute := make(map[string][]*typeModel)
	for i := range graph.Nodes {
		node := &graph.Nodes[i]
		if node.Classification != pythonimports.Type || node.CanonicalName.Empty() {
			continue
		}
		t := newTypeModel(node.ID, node.CanonicalName.String())
		for attr := range node.Members {
			typesByAttribute[attr] = append(typesByAttribute[attr], t)
			t.attrs[attr] = 1.
		}
		t.max = 1.
		types[node.ID] = t
	}

	// Load usages with observed types from dynamic analysis
	if args.DynamicUsages != "" {
		log.Println("Enumerating dynamic usages...")
		err := serialization.Decode(args.DynamicUsages, func(u *dynamicanalysis.Usage) {
			if u.ReturnedFrom == "" || u.Type == "" {
				return
			}

			log.Printf("Loaded dynamic usage for %s (-> %s)", u.ReturnedFrom, u.Type)

			// Lookup the function
			var funcID int64
			var funcname string
			if funcnode, err := graph.Find(u.ReturnedFrom); err == nil {
				funcname, _ = graph.CanonicalName(u.ReturnedFrom)
				funcID = funcnode.ID
			} else {
				log.Println("Failed to look up func from dynamic analysis:", u.ReturnedFrom)
				funcname = u.ReturnedFrom
				funcID = -int64(len(funcs)) // use a bogus negative ID - but note that the IDs are not output
			}

			f := funcs[funcID]
			if f == nil {
				f = newFuncModel(funcID, funcname)
				funcs[funcID] = f
			}

			// Lookup the type
			var typeID int64
			var typename string
			if typenode, err := graph.Find(u.Type); err == nil {
				typename, _ = graph.CanonicalName(u.Type)
				typeID = typenode.ID
			} else {
				log.Println("Failed to look up type from dynamic analysis:", u.Type)
				typename = u.Type
				typeID = -int64(len(types)) // use a bogus negative ID - but note that the IDs are not output
			}

			t := types[typeID]
			if t == nil {
				t = newTypeModel(typeID, typename)
				types[typeID] = t
			}

			// Construct the variable
			v := newVariable(f)
			v.observedType = t
			v.types[t] = 1. // probability for observed type is 1
			f.usages = append(f.usages, v)
			t.instances = append(t.instances, v)
			observedVars = append(observedVars, v)
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	// Load usages with unobserved types from github data
	var prevPkgKnown bool
	var prevKey, prevPkg string
	var numRecords, numOtherTypes, numUnknown, numAcceptedFuncs, numAcceptedTypes int

	log.Println("Enumerating static usages...")
	r := awsutil.NewEMRIterator(rr)
	for r.Next() {
		numRecords++
		if numRecords%100000 == 0 {
			log.Printf("Processed %d records (%d funcs accepted so far, of which %d unique)",
				numRecords, numAcceptedFuncs, len(funcs))
		}

		if args.StaticLimit > 0 && numRecords >= args.StaticLimit {
			break
		}

		pkg := root(r.Key())
		if pkg == prevPkg && !prevPkgKnown {
			if numRecords%10000 == 0 {
				log.Printf("   REJECT: %s (%s)", pkg, r.Key())
			}
			continue
		}

		keyIsNew := r.Key() != prevKey
		prevKey = r.Key()
		prevPkg = pkg
		_, err := graph.Find(pkg)
		prevPkgKnown = err == nil

		// Look up node for this item
		node, err := graph.Find(r.Key())
		if err != nil {
			if keyIsNew && prevPkgKnown {
				log.Printf("   REJECT: %-30s is unknown", r.Key())
			}
			numUnknown++
			continue
		}
		cn, _ := graph.CanonicalName(r.Key())
		if node.Classification != pythonimports.Function && node.Classification != pythonimports.Type {
			if cn != "" && r.Key() != prevKey {
				log.Printf("   REJECT: %-40s is a %s", cn, node.Classification)
			}
			numOtherTypes++
			continue
		}

		if keyIsNew {
			log.Printf("** ACCEPT: %-40s is a %s", cn, node.Classification)
		}

		// Parse the usage information
		var usage pythoncode.ObjectUsage
		err = json.Unmarshal(r.Value(), &usage)
		if err != nil {
			log.Printf("warning: value from %s failed to decode as json: %v", args.StaticUsages, err)
			continue
		}

		if node.Classification == pythonimports.Function {
			numAcceptedFuncs++
			f := funcs[node.ID]
			if f == nil {
				f = newFuncModel(node.ID, cn)
				funcs[node.ID] = f
			}

			// create a variable to represent this usage
			v := newVariable(f)
			for _, attr := range usage.Attributes {
				v.attrs = append(v.attrs, attr.Identifier)
			}
			unobservedVars = append(unobservedVars, v)

			f.usages = append(f.usages, v)
		} else if node.Classification == pythonimports.Type {
			numAcceptedTypes++
			if t := types[node.ID]; t != nil {
				for _, item := range usage.Attributes {
					freq := t.attrs[item.Identifier] + 1
					if freq > t.max {
						t.max = freq
					}
					t.attrs[item.Identifier] = freq
				}
			}
		}
	}
	if r.Err() != nil {
		log.Fatalln(r.Err())
	}

	// Compute candidate returntypes for each function
	log.Println("Computing initial type/function table")
	logNumTypes := math.Log(float64(len(types)))
	for _, f := range funcs {
		log.Println("Computing initial return type distribution for", f.name)

		// compute number of times each attribute was accessed
		countByAttribute := make(map[string]int)
		observedTypes := make(map[int64]int)
		for _, usage := range f.usages {
			for _, attr := range usage.attrs {
				countByAttribute[attr]++
			}
			if usage.observedType != nil {
				observedTypes[usage.observedType.id]++
			}
		}

		// compute tfidf-like scores using types as "documents" and functions as "queries"
		scoresByType := make(map[int64]float64)
		for attr, count := range countByAttribute {
			ts := typesByAttribute[attr]
			idf := logNumTypes - math.Log(float64(len(ts)))
			for _, t := range ts {
				if t == nil {
					continue
				}
				// only consider types that belong to a package that is a dependency of the package
				// containing the function
				if d := deps[f.pkg]; d != nil && contains(d.Dependencies, t.pkg) {
					tf := .5 + (.5*t.attrs[attr])/t.max
					weight := tf * idf * float64(count)
					scoresByType[t.id] += weight
				}
			}
		}

		// initialize the probability distribution over return types
		typeIDs := topN(scoresByType, numTypesPerFunction)
		var staticDenom float64
		for _, typeID := range typeIDs {
			staticDenom += scoresByType[typeID]
		}

		// if dynamic analysis observed this function then assign 90% of probability mass to those types
		var dynamicDenom float64
		if len(observedTypes) > 0 {
			for _, count := range observedTypes {
				dynamicDenom += float64(count)
			}
			staticDenom /= .1  // static analysis gets 10% of initial probability mass
			dynamicDenom /= .9 // dynamic analysis gets 90% of initial probability mass
			for typeID, count := range observedTypes {
				t := types[typeID]
				f.returnTypes[t] += float64(count) / dynamicDenom
			}
		}

		if staticDenom < 1e-8 {
			staticDenom = 1.
		}
		for _, typeID := range typeIDs {
			t := types[typeID]
			f.returnTypes[t] += scoresByType[typeID] / staticDenom
		}

		// initialize the type distribution for each variable returned by this function to the
		// return type distribution for the function itself
		for _, v := range f.usages {
			if v.observedType == nil {
				for t, p := range f.returnTypes {
					v.types[t] = p
					t.instances = append(t.instances, v)
				}
			}
		}
	}

	// Remove types for which we have no data
	filteredTypes := make(map[int64]*typeModel)
	for id, t := range types {
		if len(t.instances) > 0 {
			filteredTypes[id] = t
		}
	}
	log.Printf("Kept %d of %d type models with at least one observation", len(filteredTypes), len(types))
	types = filteredTypes

	// Run EM
	for step := 0; step < args.Steps; step++ {
		// Update type distribution for each unobserved variable (E step)
		log.Println("Updating type probabilities for each variable...")
		for _, v := range unobservedVars {
			var sum float64
			for t := range v.types {
				p := v.function.returnTypes[t]
				for _, attr := range v.attrs {
					p *= (t.attrs[attr] + attrSmoothing)
				}
				v.types[t] = p
				sum += p
			}

			// normalize
			for t := range v.types {
				v.types[t] /= sum
				assertFinite(v.types[t], "P(vartype=%s | returnedfrom=%s)=NaN, sum=%f", t.name, v.function.name, sum)
			}
		}

		// Update attribute distribution for each type (M step, part 1)
		log.Println("Updating attribute probabilities for each type...")
		for _, t := range types {
			for attr := range t.attrs {
				t.attrs[attr] = attrSmoothing
			}
			for _, v := range t.instances {
				p := v.types[t]
				for _, attr := range v.attrs {
					t.attrs[attr] += p
				}
			}

			// normalize
			var sum float64
			for _, p := range t.attrs {
				sum += p
			}
			if sum < 1e-8 && len(t.attrs) > 0 {
				log.Printf("Warning: sum=%f for type %s (n=%d)", sum, t.name, len(t.attrs))
			}
			for attr := range t.attrs {
				t.attrs[attr] /= sum
				assertFinite(t.attrs[attr], "P([%s].%s)=NaN, sum=%f, num_instances=%d", t.name, attr, sum, len(t.instances))
			}
		}

		// Update return type distribution for each function (M step, part 2)
		log.Println("Updating return type probabilities for each function...")
		for _, f := range funcs {
			f.returnTypes = make(map[*typeModel]float64)
			for _, v := range f.usages {
				for t, p := range v.types {
					f.returnTypes[t] += p
				}
			}

			// normalize
			var sum float64
			for _, p := range f.returnTypes {
				sum += p
			}
			if sum < 1e-8 {
				log.Printf("Warning: sum=%f for function %s (n=%d)", sum, f.name, len(f.returnTypes))
			}
			for t := range f.returnTypes {
				f.returnTypes[t] /= sum
				assertFinite(f.returnTypes[t], "P(%s | %s)=NaN, sum=%f, num_return_types=%d", t.name, f.name, sum, len(f.returnTypes))
			}
		}
	}

	// Compute final function models
	for _, f := range funcs {
		model := typeinduction.Function{
			Name: f.name,
		}
		for t, p := range f.returnTypes {
			if p < 1e-8 {
				log.Printf("Warning: P(%s | %s) = %f", t.name, f.name, p)
			}
			model.ReturnType = append(model.ReturnType, typeinduction.Element{
				Name:           t.name,
				LogProbability: math.Log(p),
			})
		}
		sort.Sort(sort.Reverse(byProbability(model.ReturnType)))
		funcEnc.Encode(&model)

		log.Println(model.Name)
		for _, ret := range model.ReturnType {
			log.Printf("  %5.1f%% %s", math.Exp(ret.LogProbability)*100., ret.Name)
		}
	}

	// Compute final type models
	for _, t := range types {
		model := typeinduction.Type{
			Name: t.name,
		}
		for attr, p := range t.attrs {
			if p < 1e-8 {
				log.Printf("Warning: P([%s].%s) = %f", t.name, attr, p)
			}
			model.Attributes = append(model.Attributes, typeinduction.Element{
				Name:           attr,
				LogProbability: math.Log(p),
			})
		}
		sort.Sort(sort.Reverse(byProbability(model.Attributes)))
		typeEnc.Encode(&model)
	}

	log.Printf("Of %d input records:\n", numRecords)
	log.Printf("  Accepted %d functions (%d unique)", numAcceptedFuncs, len(funcs))
	log.Printf("  Accepted %d types (%d unique)", numAcceptedTypes, len(types))
	log.Printf("  Ignored %d unknown names", numUnknown)
	log.Printf("  Ignored %d of other categories", numOtherTypes)
}
