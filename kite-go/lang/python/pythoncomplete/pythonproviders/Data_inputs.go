package pythonproviders

import (
	"go/token"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonhelpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/local-pipelines/mixing"
)

// Global contains resources that are specific to a file, but are not tied to a specific Buffer (i.e. file contents)
type Global struct {
	ResourceManager    pythonresource.Manager
	Models             *pythonmodels.Models
	Product            licensing.ProductGetter
	SetCTATargetBuffer func(data.Buffer)
	UserID             int64
	MachineID          string
	FilePath           string
	LocalIndex         *pythonlocal.SymbolIndex
	Lexical            lexicalproviders.Global
	Normalizer         mixing.Normalizer
}

// Inputs encapsulates parsed/analyzed inputs to a Provider
type Inputs struct {
	data.SelectedBuffer

	words    []pythonscanner.Word
	resolved *pythonanalyzer.ResolvedAST
	underPos []pythonast.Node

	GGNNPredictor     *pythongraph.PredictorNew
	UsePartialDecoder bool
}

// NewInputsFromPyCtx computes inputs from a *python.Context
func NewInputsFromPyCtx(ctx kitectx.Context, g Global, b data.SelectedBuffer, allowValueMutation bool, pyctx *python.Context, usePartialDecoder bool) (Inputs, error) {
	inp := Inputs{
		SelectedBuffer:    b,
		words:             pyctx.IncrLexer.Words(),
		resolved:          pyctx.Resolved,
		UsePartialDecoder: usePartialDecoder,
	}

	var nodesUnderPos []pythonast.Node
	pythonhelpers.InspectContainingSelection(ctx, inp.resolved.Root, int64(b.Selection.Begin), int64(b.Selection.End), func(node pythonast.Node) bool {
		nodesUnderPos = append(nodesUnderPos, node)
		return true
	})
	inp.underPos = nodesUnderPos

	return inp, nil
}

// NewInputs computes inputs from a SelectedBuffer
func NewInputs(ctx kitectx.Context, g Global, b data.SelectedBuffer, allowValueMutation bool, usePartialDecoder bool) (Inputs, error) {
	inp := Inputs{
		SelectedBuffer:    b,
		UsePartialDecoder: usePartialDecoder,
	}

	src := []byte(b.Buffer.Text())
	// lex
	lexOpts := pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
		Label:        g.FilePath,
	}
	inp.words, _ = pythonscanner.Lex(src, lexOpts)
	// parse
	parseOpts := pythonparser.Options{Approximate: true}
	if b.Selection.Begin == b.Selection.End {
		cursor := token.Pos(b.Selection.Begin)
		parseOpts.Cursor = &cursor
	}
	mod, err := pythonparser.ParseWords(ctx, src, inp.words, parseOpts)
	if mod == nil {
		return Inputs{}, errors.Wrapf(err, "no module returned from pythonparser.Parse")
	}

	// analyze file
	var localGraph *pythonenv.SourceTree
	var pythonPaths map[string]struct{}
	if g.LocalIndex != nil {
		localGraph = g.LocalIndex.SourceTree
		pythonPaths = g.LocalIndex.PythonPaths
	}
	importer := pythonstatic.Importer{
		Path:        g.FilePath,
		PythonPaths: pythonPaths,
		Global:      g.ResourceManager,
		Local:       localGraph,
	}
	resolver := pythonanalyzer.NewResolverUsingImporter(importer, pythonanalyzer.Options{
		User:    g.UserID,
		Machine: g.MachineID,
		Path:    g.FilePath,
	})

	resolved, err := resolver.ResolveContext(ctx, mod, allowValueMutation)
	if err != nil {
		return Inputs{}, errors.Wrapf(err, "could not compute resolved AST")
	}

	inp.resolved = resolved

	// compute nodes under b.Selection
	var nodesUnderPos []pythonast.Node
	pythonhelpers.InspectContainingSelection(ctx, resolved.Root, int64(b.Selection.Begin), int64(b.Selection.End), func(node pythonast.Node) bool {
		nodesUnderPos = append(nodesUnderPos, node)
		return true
	})
	inp.underPos = nodesUnderPos

	return inp, nil
}

// Words returns lexed Words for the Buffer
func (i Inputs) Words() []pythonscanner.Word {
	return i.words
}

// ResolvedAST returns a parsed & analyzed ResolvedAST
func (i Inputs) ResolvedAST() *pythonanalyzer.ResolvedAST {
	return i.resolved
}

// UnderSelection returns all AST Nodes fully containing the buffer Selection
func (i Inputs) UnderSelection() []pythonast.Node {
	return i.underPos
}
