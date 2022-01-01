package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
)

const (
	// The number of lines that were added to every line index in the AST
	// as a result of placing the snippet within a type tracing boilerplate
	// before AST construction.
	boilerplateLines = 5
)

// Pos is a pair of line number and column offset.
type Pos struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

// Expression contains information about a single expression.
type Expression struct {
	Expression     string `json:"expression"`
	FullyQualified string `json:"fully_qualified"`
	Begin          Pos    `json:"begin"`
	End            Pos    `json:"end"`
}

// NameAtRuntimeCollated contains information about all expressions for a snapshot collated.
type NameAtRuntimeCollated struct {
	SnapshotID  int64 `json:"snapshot_id"`
	Expressions []Expression
}

// NameAtRuntime contains information about an expression's fully qualified name at runtime.
type NameAtRuntime struct {
	SnippetID      int64  `json:"snippet_id"`      // ID of snippet
	Expression     string `json:"expression"`      // For example, 'os.path.join'. As it appears in the snippet.
	FullyQualified string `json:"fully_qualified"` // Fully qualified function name that the expression is interpreted as.
	Begin          Pos    `json:"begin"`           // Signifies start of function call in source code
	End            Pos    `json:"end"`             // Signifies last argument of function call in source code
}

func main() {
	var input, output string
	var collate bool
	flag.StringVar(&input, "input", "", "json.gz file containing output of runtime type inference (ASTs in json format)")
	flag.BoolVar(&collate, "collate", false, "output NameAtRuntimeCollated if set to true")
	flag.StringVar(&output, "output", "", "json.gz file that will contain map of code to fqn")
	flag.Parse()

	if input == "" {
		log.Fatalf("Please specify input file name.")
	}

	decoder := dynamicanalysis.NewDecoder(input)

	var fullyQualifiedNames []interface{}
	for {
		dat := make(map[string]interface{})
		err := decoder.Decode(&dat)
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				log.Println("Encountered unexpected EOF")
				break
			} else if err == io.EOF {
				log.Println("Reached EOF. Exit gracefully.")
				break
			} else {
				log.Fatal(err)
				break
			}
		}

		// SnippetID of current snippet
		snippetID := dat["SnippetID"]
		snippetIDFloat, ok := snippetID.(float64)
		if !ok {
			log.Printf("SnippetID should be float64\n")
			continue
		}

		var collated NameAtRuntimeCollated
		if collate {
			// SnapshotID of current snippet
			snapshotID := dat["SnapshotID"]
			snapshotIDFloat, ok := snapshotID.(float64)
			if !ok {
				log.Printf("SnapshotID should be float64\n")
				continue
			}

			collated = NameAtRuntimeCollated{
				SnapshotID:  int64(snapshotIDFloat),
				Expressions: []Expression{},
			}
		}

		// Get all Call expressions from AST
		var callExprs []interface{}
		dynamicanalysis.CollectValuesForKey("Call", dat["RootArray"], &callExprs)

		for _, call := range callExprs {
			callMap, ok := call.(map[string]interface{})
			if !ok {
				log.Println("'Call' expression should be a map.")
				continue
			}

			log.Println("Deducing original expression...")
			expression, err := deduceOriginalExpression(callMap)
			if err != nil {
				log.Println(err)
				continue
			}

			log.Println("Finding fully qualified name...")
			fqn, err := dynamicanalysis.FindFullyQualifiedName(callMap)
			if err != nil {
				log.Println(err)
				continue
			}

			log.Println("Finding line and column of beginning of invocation...")
			lineNumberOfBeginning, colOffsetOfBeginning, err := findInvocationBeginLineAndCol(callMap)
			if err != nil {
				log.Println(err)
				continue
			}

			log.Println("Deducing line and column of end of invocation...")
			lineNumberOfEnd, colOffsetOfEnd, err := deduceInvocationEndLineAndCol(callMap)
			if err != nil {
				log.Println(err)
				continue
			}

			// Put together all the data to output to json
			if !collate {
				fullyQualifiedName := NameAtRuntime{
					SnippetID:      int64(snippetIDFloat),
					Expression:     expression,
					FullyQualified: fqn,
					Begin: Pos{
						Line: lineNumberOfBeginning,
						Col:  colOffsetOfBeginning,
					},
					End: Pos{
						Line: lineNumberOfEnd,
						Col:  colOffsetOfEnd,
					},
				}
				fullyQualifiedNames = append(fullyQualifiedNames, fullyQualifiedName)
			} else {
				collated.Expressions = append(collated.Expressions, Expression{
					Expression:     expression,
					FullyQualified: fqn,
					Begin: Pos{
						Line: lineNumberOfBeginning,
						Col:  colOffsetOfBeginning,
					},
					End: Pos{
						Line: lineNumberOfEnd,
						Col:  colOffsetOfEnd,
					},
				})
			}

			log.Printf("Expression %s is interpreted as a call to %s\n", expression, fqn)
		}

		if collate {
			fullyQualifiedNames = append(fullyQualifiedNames, collated)
		}
	}

	// create a file for output
	fout, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	// create a compressor and an encoder
	comp := gzip.NewWriter(fout)
	enc := json.NewEncoder(comp)
	defer comp.Close()

	enc.Encode(fullyQualifiedNames)
	log.Printf("Num expressions evaluated: %d\n", len(fullyQualifiedNames))
}

// --

// Combine the values held in the 'id' and 'attr' fields to re-construct the original
// expression. This function needs to be extended to account for nested Call expressions
// that are generated by call chains like 'numpy.array([...]).transpose()'. Currently,
// that would evaluate to 'numpy.array.transpose' instead.
func deduceOriginalExpression(callMap map[string]interface{}) (string, error) {
	// Reverse engineer expression from AST

	// The 'id' key is a descendent of a 'Call' object
	// It contains the name of the module/package/subpackage on which the 'Call' expression operates
	// For example, for 'numpy.array()', the 'id' key would map to 'numpy'
	ids, err := dynamicanalysis.CollectStringsForKey("id", callMap["func"])
	if err != nil {
		return "", fmt.Errorf("error while type asserting ids to strings: %v", err)
	}
	if len(ids) != 1 { // There can only be one module/package/subpackage that a 'Call' expression is operating on.
		return "", fmt.Errorf("there should be exactly 1 id field in a 'Call' expression")
	}

	// The 'attr' key(s) contain the remaining subpackage names that help locate the function within the 'id'.
	// For example, for 'os.path.join()', there will be two 'attr' descendents of the 'Call' object, one
	// containing 'path' and the other 'join'. The 'id' will be 'os'.
	attrs, err := dynamicanalysis.CollectStringsForKey("attr", callMap["func"])
	if err != nil {
		return "", fmt.Errorf("error while type asserting attrs to strings: %v", err)
	}

	// Put together the expression as written by the user from the id
	// and the attrs
	code := strings.Join(append(ids, attrs...), ".")
	return code, nil
}

// Note that BOILERPLATE_LINES is subtracted from the line number of the
// invocation to account for placing the snippet within a type tracing
// boilerplate before AST construction.
func findInvocationBeginLineAndCol(callMap map[string]interface{}) (int, int, error) {
	// The AST is traversed using DFS so we take the last element to get
	// the line/col that is closest to the root. This gives us the line and col
	// of the invocation of the function.

	// Get line where function was invoked
	lineNumbers, err := collectFloatsForKey("k_lineno", callMap["func"])
	if err != nil {
		return -1, -1, fmt.Errorf("error collecting line numbers in Call expression: %v", err)
	}
	if len(lineNumbers) == 0 {
		return -1, -1, fmt.Errorf("there should be at least one line number of the Call expression")
	}
	lineNumberOfInvocation := int(lineNumbers[len(lineNumbers)-1]) - boilerplateLines // subtract because of tracing boilerplate

	// Get column offset where function was invoked
	columnOffsets, err := collectFloatsForKey("k_col_offset", callMap["func"])
	if err != nil {
		return -1, -1, fmt.Errorf("error collecting column offsets in Call expression: %v", err)
	}
	if len(columnOffsets) == 0 {
		return -1, -1, fmt.Errorf("there should be at least one column offset in Call expression")
	}
	colOffsetOfInvocation := int(columnOffsets[len(columnOffsets)-1])

	return lineNumberOfInvocation, colOffsetOfInvocation, nil
}

// The end of the invocation is approximated by the line and col offset of the
// first character of the last argument. Where possible to obtain, the actual
// col offset of the final paranthesis of the invocation is returned. Note that
// BOILERPLATE_LINES is subtracted from the line number of the last argument to
// account for placing the snippet within a type tracing boilerplate before AST
// construction.
func deduceInvocationEndLineAndCol(callMap map[string]interface{}) (int, int, error) {
	// Get line number of last argument to invocation
	lineNumberOfLastArg := -1
	lineNumbers, err := collectFloatsForKey("k_lineno", callMap["args"])
	if err != nil {
		return -1, -1, fmt.Errorf("error collecting line numbers of args in Call expression: %v", err)
	}
	if len(lineNumbers) > 0 {
		lineNumberOfLastArg = int(lineNumbers[len(lineNumbers)-1]) - boilerplateLines // subtract because of tracing boilerplate
	}

	// Get column offset of first character of last argument to invocation
	colOffsetOfLastArg := -1
	columnOffsets, err := collectFloatsForKey("k_col_offset", callMap["args"])
	if err != nil {
		return -1, -1, fmt.Errorf("error collecting column offsets of args in Call expression: %v", err)
	}
	if len(columnOffsets) > 0 {
		colOffsetOfLastArg = int(columnOffsets[len(columnOffsets)-1])

		// Try to get name of last arg so can point to after the last arg instead
		if args, err := dynamicanalysis.CollectStringsForKey("s", callMap["args"]); err == nil {
			// If they are not the same length, we can't be sure that
			// the last arg name corresponds to the last col offset
			// In these cases, just leave the end col offset at the
			// beginning of the last argument
			if len(args) == len(columnOffsets) {
				lastArg := args[len(args)-1]

				// Add the length of the arg + 1 (for trailing paranthesis)
				// so that we get the ending offset of the entire function call
				colOffsetOfLastArg += len(lastArg) + 1
			}
		}
	}

	return lineNumberOfLastArg, colOffsetOfLastArg, nil
}

func sliceOfFloats(original []interface{}) ([]float64, error) {
	var floats []float64
	for i, elem := range original {
		elemFloat, ok := elem.(float64)
		if !ok {
			return nil, fmt.Errorf("element %d is not float but %v", i, elem)
		}
		floats = append(floats, elemFloat)
	}
	return floats, nil
}

func constructUniqueKey(snippetID int64, code string) string {
	return fmt.Sprintf("%d-%s-%d", snippetID, code, rand.Int())
}

func collectFloatsForKey(key string, root interface{}) ([]float64, error) {
	var buf []interface{}
	dynamicanalysis.CollectValuesForKey(key, root, &buf)
	return sliceOfFloats(buf)
}
