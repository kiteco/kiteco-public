package dynamicanalysis

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/sandbox"
)

var (
	// Load the code for python-side tracing
	harnessCode = MustAsset("src/trace_references.py")
)

// TraceReferences executes the given python code and returns references associating
// tokens in the source file with their fully qualified names.
func TraceReferences(src string, opts TraceOptions) ([]Reference, *sandbox.Result, error) {
	// Construct the program
	prog := sandbox.NewContainerizedPythonProgram(string(harnessCode), opts.DockerImage)
	prog.SupportingFiles["src.py"] = []byte(src)
	prog.EnvironmentVariables["PYTHONPATH"] = "."
	prog.EnvironmentVariables["SOURCE"] = "src.py"
	prog.EnvironmentVariables["TRACE_OUTPUT"] = "references.json"

	// Construct the apparatus
	apparatus, err := annotate.NewApparatusFromCode(src, lang.Python)
	apparatus.SetLimits(&opts.Limits)
	if err != nil {
		return nil, nil, fmt.Errorf("error constructing apparatus: %v", err)
	}

	// Run the program in the apparatus
	result, err := apparatus.Run(prog)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute tracing code: %v", err)
	} else if !result.Succeeded {
		return nil, nil, fmt.Errorf("tracing code exited uncleanly: %v\nPython said:\n%s", result.SandboxError, result.Stderr)
	} else if len(result.Stderr) > 0 {
		// Stderr is not empty but the process exited with status 0. In this case it may be
		// helpful to print what was received on stderr, but we should still continue anyway.
		log.Printf("Stderr from tracing: %s\n", string(result.Stderr))
	}

	// Look for the output file
	f := result.File("references.json")
	if f == nil {
		return nil, nil, fmt.Errorf("tracing code did not generate references.json")
	}

	// Parse the references
	var references []Reference
	if err := json.Unmarshal(f.Contents, &references); err != nil {
		return nil, nil, fmt.Errorf("error parsing references: %v", err)
	}

	return references, result, nil
}

// TraceSnippetReferences executes the given code example and returns references associating
// tokens in the source file with their fully qualified names. It differs from TraceReferences
// in that it makes the presentation API and sample files available to the code example.
func TraceSnippetReferences(snippet *curation.CuratedSnippet, opts TraceOptions) ([]Reference, *annotate.Flow, error) {
	// Get individual regions
	regions := curation.RegionsFromSnippet(snippet)
	if len(regions) == 0 {
		return nil, nil, errors.New("RegionsFromSnippet returned zero regions")
	}

	// Parse spec from postlude
	spec, err := annotate.NewSpecFromCode(lang.Python, snippet.Postlude)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing apparatus spec: %v", err)
	}

	// Run with the spec
	spec.Env["TRACE_OUTPUT"] = "references.json"
	flow, err := annotate.RunWithSpec(regions, spec, annotate.Options{
		Language:    lang.Python,
		DockerImage: opts.DockerImage,
		Entrypoint:  harnessCode,
	})
	result := flow.Raw
	if err != nil {
		return nil, nil, fmt.Errorf("error executing code example: %v", err)
	} else if !result.Succeeded {
		return nil, nil, fmt.Errorf("tracing code exited uncleanly: %v\nPython said:\n%s", result.SandboxError, result.Stderr)
	} else if len(result.Stderr) > 0 {
		// Stderr is not empty but the process exited with status 0. In this case it may be
		// helpful to print what was received on stderr, but we should still continue anyway.
		log.Printf("Stderr from tracing: %s\n", string(result.Stderr))
	}

	// Look for the output file
	f := flow.Raw.File("references.json")
	if f == nil {
		return nil, nil, errors.New("tracing code did not generate references.json")
	}

	// Parse the references
	var references []Reference
	if err := json.Unmarshal(f.Contents, &references); err != nil {
		return nil, nil, fmt.Errorf("error parsing references: %v", err)
	}

	return references, flow, nil
}

// A Reference to a fully-qualified name
type Reference struct {
	Begin              int         `json:"begin"`
	End                int         `json:"end"`
	Original           string      `json:"expression"`
	FullyQualifiedName string      `json:"fully_qualified"`
	Instance           bool        `json:"instance"`
	Node               interface{} `json:"node"`
	NodeType           string      `json:"node_type"` // "import", "name", "attribute", or "call"
}

// Length returns the length of the expression
func (r Reference) Length() int {
	return r.End - r.Begin
}

// ByBeginEnd is used for sorting Reference objects by begin and end indices
type ByBeginEnd []Reference

func (a ByBeginEnd) Len() int      { return len(a) }
func (a ByBeginEnd) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBeginEnd) Less(i, j int) bool {
	return a[i].Begin < a[j].Begin ||
		(a[i].Begin == a[j].Begin && a[i].End < a[j].End)
}

// SortReferencesByBeginEnd sorts an array of References in ascending order
// using Begin first, then End
func SortReferencesByBeginEnd(refs []Reference) {
	sort.Sort(ByBeginEnd(refs))
}

// LineIndexedReference contains a Reference along with line number and column
// offset indices
type LineIndexedReference struct {
	Reference  Reference
	LineNumber int // 0-indexed
	ColOffset  int
}

// ByLineNumber is used for sorting LineIndexedReference objects by line
// number and column offset
type ByLineNumber []LineIndexedReference

func (a ByLineNumber) Len() int      { return len(a) }
func (a ByLineNumber) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLineNumber) Less(i, j int) bool {
	return a[i].LineNumber < a[j].LineNumber ||
		(a[i].LineNumber == a[j].LineNumber && a[i].ColOffset < a[j].ColOffset)
}

// SortReferencesByLineNumber sorts an array of LineIndexedReferences in
// ascending order using LineNumber first, then ColOffset
func SortReferencesByLineNumber(indexed []LineIndexedReference) {
	sort.Sort(ByLineNumber(indexed))
}

// GetLineIndexed finds the line number and column offset indices for a list
// of References and returns a list of LineIndexedReference objects. This
// function assumes that the input source was the actual source code that
// generated the input list of References
func GetLineIndexed(refs []Reference, source string) []LineIndexedReference {
	var indexed []LineIndexedReference
	SortReferencesByBeginEnd(refs)
	lines := strings.Split(source, "\n")
	count := []int{0}
	for i := 0; i < len(lines)-1; i++ {
		count = append(count, count[len(count)-1]+len(lines[i])+1)
	}

	var line int
	for _, ref := range refs {
		for ; line < len(count); line++ {
			if count[line] <= ref.Begin && (line == len(count)-1 || count[line+1] > ref.Begin) {
				indexed = append(indexed, LineIndexedReference{
					Reference:  ref,
					LineNumber: line,
					ColOffset:  ref.Begin - count[line],
				})
				break
			}
		}
	}

	return indexed
}

// ResolvedSnippet represents python code with a set of references
// A ResolvedSnippet may contain a reference (in the form of a numeric ID) to
// an external code snippet object
type ResolvedSnippet struct {
	Code          string
	PresentedCode string `json:"PresentedCode,omitempty"`
	LineMap       []int  `json:"LineMap,omitempty"`
	References    []Reference
	SnippetID     int64 `json:"SnippetID,omitempty"`
}
