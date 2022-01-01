package helpers

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

const (
	// MaxTypes defines the maximum number of types kept for an argument in a signature example
	MaxTypes = 4
	// MaxExamples defines the maximum number of examples for each argument in a editorAPI.signature
	MaxExamples = 3
)

// transformers
var exampleSanitizer = regexp.MustCompile("\n\\s*")

func transformExamples(argType string, values []*pythoncode.StringCount, maxExamples int) []string {
	if len(values) > maxExamples {
		values = values[:maxExamples]
	}

	var ret []string
	for _, val := range values {
		if argType == "builtins.str" {
			ret = append(ret, string(exampleSanitizer.ReplaceAll([]byte(fmt.Sprintf("\"%s\"", val.Value)), []byte(""))))
		} else {
			ret = append(ret, string(exampleSanitizer.ReplaceAll([]byte(val.Value), []byte(""))))
		}
	}
	return ret
}

// builtin types really live in the `types` module; eventually we should fix canonical name computation to avoid having to do this
var typeRenames = map[string]string{
	"builtins.builtin_function_or_method": "six.types.BuiltinFunctionType",
	"builtins.classobj":                   "builtins.type",
	"builtins.function":                   "types.FunctionType",
	"builtins.instancemethod":             "types.MethodType",
	"builtins.module":                     "types.ModuleType",
	"builtins.NotImplementedType":         "types.NotImplemented.__class__",
}

func transformArg(graph *pythonimports.Graph, rm pythonresource.Manager, index Compat, arg *pythoncode.ArgStats, maxExamples, maxTypes int) *editorapi.ParameterExample {
	if arg.Name == "" {
		return nil
	}

	var types []*editorapi.ParameterTypeExample

	if arg.Type != "" {
		for _, t := range strings.Split(arg.Type, ",") {
			if len(types) >= maxTypes {
				break
			}

			examples := arg.LiteralsByType[t]
			if newT, exists := typeRenames[t]; exists { // rename some types because our canonical names are wrong
				t = newT
			}

			if t != "..." { // allow "..." as a type for historical reasons
				path := pythonimports.NewDottedPath(t)

				// first try directly looking up in symbol graph
				sym, err := rm.PathSymbol(path)

				// if that didn't work, try navigating in import graph and looking up in compat index
				if err != nil {
					if n, navErr := graph.Navigate(path); navErr == nil {
						sym, err = index.Lookup(rm, n.ID)
					}
				}

				if err != nil {
					log.Printf("type %s not found in symbol graph\n", t)
					continue
				}

				// use the actual path of the symbol, in case we got it from the compat index
				t = sym.Canonical().PathString()
			}

			types = append(types, &editorapi.ParameterTypeExample{
				ID:       editorapi.NewID(lang.Python, t),
				Name:     t[strings.LastIndex(t, ".")+1:],
				Examples: transformExamples(t, examples, maxExamples),
			})
		}
	}

	return &editorapi.ParameterExample{
		Name:  arg.Name,
		Types: types,
	}
}

// TransformPattern converts a pythoncode.SignaturePattern to an editorapi.Signature that will be served
// in the resource manager API
func TransformPattern(graph *pythonimports.Graph, rm pythonresource.Manager, index Compat, pat *pythoncode.SignaturePattern, maxExamples, maxTypes int) *editorapi.Signature {
	var outArgs []*editorapi.ParameterExample
	var outKwargs []*editorapi.ParameterExample

	for _, arg := range pat.PrivateArgs() {
		outArg := transformArg(graph, rm, index, arg, maxExamples, maxTypes)
		if outArg == nil {
			return nil
		}
		outArgs = append(outArgs, outArg)
	}

	kwargs := pat.PrivateKwargs()
	seen := make(map[string]bool)
	for _, name := range pat.Kwargs {
		if _, exists := seen[name]; exists {
			continue
		}
		if kwarg, ok := kwargs[name]; ok {
			outKwargs = append(outKwargs, transformArg(graph, rm, index, kwarg, maxExamples, maxTypes))
			seen[name] = true
		}
	}

	return &editorapi.Signature{
		Args: outArgs,
		LanguageDetails: editorapi.LanguageSignatureDetails{
			Python: &editorapi.PythonSignatureDetails{Kwargs: outKwargs},
		},
	}
}
