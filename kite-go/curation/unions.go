package curation

import (
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/curation/segment"
)

// Example is the structure that is dumped to a static file and served up live from user node.
// It is different to CuratedSnippet because it contains references to other database records
// whereas CuratedSnippet contains just the fields from the respective database table.
type Example struct {
	Snippet *CuratedSnippet
	Result  *ExecutionResult
}

// A Reference to a fully-qualified name. This data type conceptually contains the same
// information as the python.Reference type, but is relevant to an individual code
// segment only. Therefore, the begin and end indices differ
type Reference struct {
	Begin              int    `json:"begin"`
	End                int    `json:"end"`
	Original           string `json:"expression"`
	FullyQualifiedName string `json:"fully_qualified"`
	Instance           bool   `json:"instance"`
	NodeType           string `json:"node_type"` // "import", "name", "attribute", or "call"
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

// ExecutionResult keeps pointers to the various DB records associated with a single execution
// of a code example
type ExecutionResult struct {
	Run         *Run
	OutputFiles []*OutputFile
	HTTPOutputs []*HTTPOutput
	Problems    []*CodeProblem
	Segments    []*Segment
	InputFiles  []*annotate.InputFile
}

// String gets a text representation of the result
func (r *ExecutionResult) String() string {
	// Append stdout and stderr
	output := string(r.Run.Stdout)
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	output += string(r.Run.Stderr)

	// Append HTTP outputs
	for _, httpoutput := range r.HTTPOutputs {
		// Make sure there is an empty line before each HTTP response
		for !strings.HasSuffix(output, "\n\n") {
			output += "\n"
		}

		output += httpoutput.ResponseStatus + "\n"
		output += httpoutput.ResponseHeaders + "\n"
		if len(httpoutput.ResponseBody) > 0 {
			output += "\n"
			output += string(httpoutput.ResponseBody)
		}
		if !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
	}

	// Append error message
	if r.Run.SandboxError != "" {
		// Make sure there is an empty line before the error message
		for !strings.HasSuffix(output, "\n\n") {
			output += "\n"
		}
		output += r.Run.SandboxError + "\n"
	}
	return output
}

// References returns a list of all the references to fully-qualified names
// in a single execution run
func (r *ExecutionResult) References() []Reference {
	var refs []Reference
	for _, s := range r.Segments {
		if s.Type == segment.Code {
			for _, r := range s.References {
				refs = append(refs, r)
			}
		}
	}
	return refs
}
