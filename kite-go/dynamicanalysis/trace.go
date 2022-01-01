//go:generate go-bindata -pkg dynamicanalysis -o bindata.go src/...

package dynamicanalysis

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/sandbox"
)

const (
	// The path to which the ast will be written when the code is executed
	astPath = "tree.json"
)

var (
	// The code containing the entrypoint
	mainCode = MustAsset("src/run.py")
	// The code containing the macropy decorator etc
	tracelibCode = MustAsset("src/tracelib.py")
	// DefaultTraceOptions is a set of default options
	DefaultTraceOptions = TraceOptions{
		DockerImage: "kiteco/pythonsandbox",
		Limits:      sandbox.Limits{Timeout: 3 * time.Second},
	}

	// This is inserted above the code that is to be instrumented
	boilerplateStart = fmt.Sprintf(`
import sys
import re
import tracelib
from tracelib import macros, kite_trace, get_all_traced_ast_reprs

def callback():
    with open("%s", "w") as f:
        f.write('\n\n'.join(get_all_traced_ast_reprs(
            indent='  ',
            include_field_names=True)))

tracelib.callback = callback

with kite_trace:
`, astPath)

	// Delimiters that designate the beginning and ending of a multi line string in Python
	multiLineStringDelims = []string{`"""`, `'''`}
)

// Tree is a raw unmarshal of the json tree generating by the macropy tracing code.
type Tree map[string]interface{}

// TraceOptions represents options for running the dynamic tracer
type TraceOptions struct {
	DockerImage string
	Limits      sandbox.Limits
}

// TraceResult encapsulates the full result of running code through dynamic analysis
type TraceResult struct {
	ModifiedCode string
	Tree         Tree
	Output       *sandbox.Result
}

// Trace executes the given code in a sandbox, and uses a set of macropy transforms to output an AST
// annotated with type information.
func Trace(src string, opts TraceOptions) (*TraceResult, error) {
	code := indentCode(src)

	// Construct the program
	prog := sandbox.NewContainerizedPythonProgram(string(mainCode), opts.DockerImage)
	prog.SupportingFiles["snippet.py"] = []byte(code)
	prog.SupportingFiles["tracelib.py"] = tracelibCode
	prog.EnvironmentVariables["PYTHONPATH"] = "."

	// Run the program
	result, err := runPythonCodeInstrumented(src, prog)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tracing code: %v", err)
	} else if !result.Succeeded {
		return nil, fmt.Errorf("snippet failed to execute: %v\nPython said:\n%s", result.SandboxError, result.Stderr)
	} else if len(result.Stderr) > 0 {
		// Stderr is not empty but the process exited with status 0. In this case it may be
		// helpful to print what was received on stderr, but we should still continue anyway.
		log.Printf("Stderr from docker container: %s\n", string(result.Stderr))
	}

	// Look for the output file
	f := result.File(astPath)
	if f == nil {
		return nil, fmt.Errorf("tracing code did not generate %s", astPath)
	}

	// Parse the ast
	var tree Tree
	if err := json.Unmarshal(f.Contents, &tree); err != nil {
		return nil, fmt.Errorf("error '%v' unmarshalling with AST: %s", err, string(f.Contents))
	}

	trace := &TraceResult{
		ModifiedCode: code,
		Tree:         tree,
		Output:       result,
	}
	return trace, nil
}

// runPythonCodeInstrumented runs the given instrumentation code on the given program.
func runPythonCodeInstrumented(code string, prog sandbox.Program) (*sandbox.Result, error) {
	apparatus, err := annotate.NewApparatusFromCode(code, lang.Python)
	apparatus.SetLimits(&sandbox.Limits{Timeout: 3 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("error constructing apparatus: %v", err)
	}

	return apparatus.Run(prog)
}

func indentCode(src string) string {
	// Wrap snippet code within tracing boilerplate
	code := boilerplateStart

	// Indent all code lines by a tab because it will be placed within a "with" statement below
	code += "\t" + strings.Replace(src, "\n", "\n\t", -1)

	for _, delim := range multiLineStringDelims {
		code = resolveMultiLineStrings(code, delim)
	}

	return code
}

// resolveMultiLineStrings removes the leading tab from all lines within a multi line string.
func resolveMultiLineStrings(code string, delim string) string {
	var modified string
	var block bool
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		if block {
			line = strings.TrimPrefix(line, "\t")
		}
		modified += line + "\n"
		if strings.Contains(line, delim) {
			block = !block
		}
	}
	return modified
}
