package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// transformers
var exampleSanitizer = regexp.MustCompile("\n\\s*")

var strSymHashes = map[pythonimports.Hash]bool{
	pythonimports.NewDottedPath("builtins.str").Hash: true,
}

var forbiddenIdent = map[string]bool{
	"None":  true,
	"self":  true,
	"True":  true,
	"False": true,
}

type strCount struct {
	Str   string
	Count int
}

func newStrCounts(strs pythonpatterns.StrCount) []strCount {
	var values []strCount
	for s, c := range strs {
		values = append(values, strCount{
			Str:   s,
			Count: c,
		})
	}

	sort.Slice(values, func(i, j int) bool {
		if values[i].Count != values[j].Count {
			return values[i].Count > values[j].Count
		}

		// If there's a tie with the count, we use length and alphabetical order to have a deterministic process
		// That's define that the best length for a arg name is 3, that allows to avoid having too much 1 char names
		iLen := math.Abs(float64(len(values[i].Str) - 3))
		jLen := math.Abs(float64(len(values[j].Str) - 3))
		if iLen != jLen {
			return iLen < jLen
		}
		return values[i].Str < values[j].Str
	})
	return values
}

func transformExamples(s pythonresource.Symbol, strs pythonpatterns.StrCount) []string {
	var ret []string
	for _, val := range newStrCounts(strs) {
		if strSymHashes[s.PathHash()] {
			ret = append(ret, string(exampleSanitizer.ReplaceAll([]byte(fmt.Sprintf("\"%s\"", val.Str)), []byte(""))))
		} else {
			ret = append(ret, string(exampleSanitizer.ReplaceAll([]byte(val.Str), []byte(""))))
		}
	}

	if len(ret) > maxExamples {
		ret = ret[:maxExamples]
	}

	return ret
}

// builtin types really live in the `types` module; eventually we should fix canonical name computation to avoid having to do this
var typeRenames = map[string]string{
	"builtins.builtin_function_or_method": "types.BuiltinFunctionType",
	"builtins.classobj":                   "types.ClassType",
	"builtins.function":                   "types.FunctionType",
	"builtins.instancemethod":             "types.MethodType",
	"builtins.module":                     "types.ModuleType",
	"builtins.NotImplementedType":         "types.NotImplementedType",
}

func transformArg(arg *argument) *editorapi.ParameterExample {
	type hashCount struct {
		Hash  pythonimports.Hash
		Count int
	}

	var hashCounts []hashCount
	for h, sc := range arg.Types {
		hashCounts = append(hashCounts, hashCount{
			Hash:  h,
			Count: sc.Count,
		})
	}

	sort.Slice(hashCounts, func(i, j int) bool {
		return hashCounts[i].Count > hashCounts[j].Count
	})

	var types []*editorapi.ParameterTypeExample
	var sum float64
	for _, hc := range hashCounts {
		if len(types) > maxTypes {
			break
		}

		sym := arg.Types[hc.Hash].Sym
		types = append(types, &editorapi.ParameterTypeExample{
			ID:       editorapi.NewID(lang.Python, sym.PathString()),
			Name:     sym.Path().Last(),
			Examples: transformExamples(sym, arg.SrcStrsByType[hc.Hash]),
		})
		sum += float64(hc.Count)
	}

	for i := range types {
		types[i].Frequency = float64(hashCounts[i].Count) / sum
	}

	name := arg.Name
	if name == "" {
		// TODO: this is pretty janky...
		// but otherwise we lose ALOT of patterns because
		// 1) the signature just has *varargs
		// 2) we do not have an argspec

		// We first try to get a name from all the expressions gathered for this parameter
		name = maxIdentKey(arg.SrcStrs)
		if name == "" {
			// Let's try with the typed expressions if we didn't found any name in arg.SrcStrs
			for _, hc := range hashCounts {
				name = maxIdentKey(arg.SrcStrsByType[hc.Hash])
				if name != "" {
					break
				}
			}
		}
	}

	if name == "" {
		return nil
	}
	return &editorapi.ParameterExample{
		Name:  name,
		Types: types,
	}
}

func transformPattern(rm pythonresource.Manager, pattern pattern) *editorapi.Signature {
	var outArgs []*editorapi.ParameterExample
	var outKwargs []*editorapi.ParameterExample

	for _, arg := range pattern.Positional {
		outArg := transformArg(arg)
		if outArg == nil {
			return nil
		}
		outArgs = append(outArgs, outArg)
	}

	for _, arg := range pattern.Keyword {
		outArg := transformArg(arg)
		if outArg == nil {
			return nil
		}
		outKwargs = append(outKwargs, outArg)
	}

	return &editorapi.Signature{
		Args: outArgs,
		LanguageDetails: editorapi.LanguageSignatureDetails{
			Python: &editorapi.PythonSignatureDetails{Kwargs: outKwargs},
		},
		Frequency: pattern.Frequency,
	}
}

func maxIdentKey(sc pythonpatterns.StrCount) string {
	for _, v := range newStrCounts(sc) {
		if _, forbidden := forbiddenIdent[v.Str]; pythonscanner.IsValidIdent(v.Str) && !forbidden {
			return v.Str
		}
	}
	return ""
}
