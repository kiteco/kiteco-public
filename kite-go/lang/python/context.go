package python

import (
	"fmt"
	"go/token"
	"io"
	"time"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonhelpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	defaultResolverTimeout = 100 * time.Millisecond
)

// Context contains information parsed out of a python file relevant
// to the user's current cursor position.
type Context struct {
	ContextInputs
	// TrimmedCursor is the cursor after moving to the nearest non-whitespace position
	// (see nearestNonWhitespace)
	TrimmedCursor int64
	// Resolved is a list of expressions in the AST that resolved to a fully qualified name
	Resolved *pythonanalyzer.ResolvedAST
	// AST is the raw AST parser from file
	AST *pythonast.Module
	// ParseErrors for the AST
	ParseErrors errors.Errors
	// BufferIndex is the code index computed online within the driver
	BufferIndex *bufferIndex
	// UnderCursor is the list of nodes that intersect the trimmed cursor position, starting
	// with the pythonast.Module and ending with the most deeply nested node.
	UnderCursor []pythonast.Node
	// ArtifactID is the ID of the local code artifact used when building this context
	ArtifactID string
	// LocalBuildTime is the timestamp of the index used when building this context
	LocalBuildTime time.Time
}

// ContextInputs represents the things that are used to construct a context
type ContextInputs struct {
	// User identifies the user this context is for
	User int64
	// Machine identifies the machine this context is for
	Machine string
	// Buffer contains the contents of the file the user is currently editing
	Buffer []byte
	// Cursor is the byte offset of the cursor
	Cursor int64
	// Selection is the byte offsets of the begin and end of the selected region
	Selection [2]int64
	// LastAction is the action field of the most recent event from the client:
	// "edit", "selection", "focus", or "surface"
	LastAction string
	// FileName is the name of the file the user is currently editing
	FileName string
	// Resolver resolves expressions to values
	Resolver *pythonanalyzer.Resolver
	// IncrLexer is the incremental lexer. If this is non-nil then the buffer will not be
	// tokenized and instead the token list from the incremental lexer will be used.
	IncrLexer *pythonscanner.Incremental
	// LocalIndex contains the index constructed by the local code worker
	LocalIndex *pythonlocal.SymbolIndex
	// Importer is responsible for looking up top-level packages in the local and global graphs
	Importer pythonstatic.Importer
	// D writes to the diagnostics panel, or is nil if diagnostics are not enabled
	D io.Writer
	// EventSource is either the name of the editor that triggered the event or kited
	EventSource string
	// ResolverTimeout defines the maximum acceptable duration for resolving the AST
	ResolverTimeout time.Duration
	// ArtifactError is any error encountered fetching a local code artifact
	ArtifactError error
}

// NewContext builds a context using the recursive descent parser,
// the returned error is non nil if the code contains any syntax errors.
func NewContext(ctx kitectx.Context, in ContextInputs) (*Context, error) {
	ctx.CheckAbort()

	if in.Cursor > int64(len(in.Buffer)) {
		return nil, fmt.Errorf("position %d is past end of buffer (len %d)",
			in.Cursor, len(in.Buffer))
	}

	// Parse Source
	// nil mod (or mod with no Body stmts) indicates we could not parse a partial (or competlete) ast.
	// non nil parseErrs indicates we encountered parser errors, but we may still
	// have been able to get a partial ast
	cursor := token.Pos(in.Cursor)
	parseOpts := pythonparser.Options{
		Approximate: true,
		Cursor:      &cursor,
	}
	parseOpts.ScanOptions.Label = in.FileName

	start := time.Now()
	mod, parseErr := pythonparser.ParseWords(ctx, in.Buffer, in.IncrLexer.Words(), parseOpts)

	parseDuration.RecordDuration(time.Since(start))
	ctx.Logger.Durations.Record("python/NewContext (parser)", time.Since(start))

	// nil mod, no partial (or complete ast)
	if mod == nil {
		return nil, parseErr
	}
	parseErrs, _ := parseErr.(errors.Errors)

	// Do static analysis
	var resolved *pythonanalyzer.ResolvedAST
	err := ctx.WithTimeout(in.ResolverTimeout, func(ctx kitectx.Context) (err error) {
		ctx.CheckAbort()

		start = time.Now()
		resolved, err = in.Resolver.ResolveContext(ctx, mod, false)
		resolveDuration.RecordDuration(time.Since(start))
		ctx.Logger.Durations.Record("python/NewContext (resolver)", time.Since(start))
		return
	})
	if err != nil {
		if _, ok := err.(kitectx.ContextExpiredError); ok {
			resolveTimeout.Hit()
		} else {
			resolveTimeout.Miss()
		}
		return nil, err
	}

	// Build the buffer index
	start = time.Now()
	var bufferIndex *bufferIndex
	if resolved != nil && resolved.Module != nil && len(in.Buffer) > 0 {
		bufferIndex = newBufferIndex(ctx, resolved, in.Buffer, in.FileName)
	}
	bufferIndexDuration.RecordDuration(time.Since(start))
	ctx.Logger.Durations.Record("python/NewContext (bufferIndex)", time.Since(start))

	// Find the nodes under the cursor
	trimmedCursor := pythonhelpers.NearestNonWhitespace(in.Buffer, in.Cursor, pythonhelpers.IsHSpace)
	nodes := pythonhelpers.NodesUnderCursor(ctx, mod, trimmedCursor)

	var artifactID string
	if in.LocalIndex != nil {
		artifactID = in.LocalIndex.ArtifactRoot
	}

	// return the context
	return &Context{
		ContextInputs: in,
		TrimmedCursor: trimmedCursor,
		Resolved:      resolved,
		AST:           mod,
		BufferIndex:   bufferIndex,
		ParseErrors:   parseErrs,
		UnderCursor:   nodes,
		ArtifactID:    artifactID,
	}, nil
}

func (d *UnifiedDriver) updateContextWithSelection(ctx kitectx.Context, pyctx *Context, evt *event.Event, cursor int64) {
	ctx.CheckAbort()

	var sel [2]int64
	if s := evt.GetSelections(); len(s) == 1 {
		sel[0], sel[1] = s[0].GetStart(), s[0].GetEnd()
		if sel[0] > sel[1] {
			sel[0], sel[1] = sel[1], sel[0]
		}
	}

	pyctx.LastAction = evt.GetAction()
	pyctx.Selection = sel
	pyctx.Cursor = cursor
	pyctx.TrimmedCursor = pythonhelpers.NearestNonWhitespace(pyctx.Buffer, pyctx.Cursor, pythonhelpers.IsHSpace)
	pyctx.UnderCursor = pythonhelpers.NodesUnderCursor(ctx, pyctx.AST, pyctx.TrimmedCursor)

	// Get the latest local code index
	localIndex := pyctx.LocalIndex

	// Early out if theres no index or its the same as one we've already used to resolve this context
	if localIndex == nil {
		haveLocalGraph.Miss()
		return
	}

	haveLocalGraph.Hit()
	localGraph := localIndex.SourceTree

	switch {
	case localIndex.LocalBuildTime.IsZero():
		// This index came from local-code-worker - compare artifact roots to determine
		// whether this index is the same as what is already loaded
		if localIndex.ArtifactRoot == pyctx.ArtifactID {
			return
		}
	default:
		// This index came from kite local. Compare build time to determine whether
		// this index should replace the existing index (currently only one index is built at a time)
		if !localIndex.LocalBuildTime.After(pyctx.LocalBuildTime) {
			return
		}
	}

	// Update artifact ID
	pyctx.ArtifactID = localIndex.ArtifactRoot
	pyctx.LocalBuildTime = localIndex.LocalBuildTime

	// Construct import environment
	importer := pythonstatic.Importer{
		PythonPaths: localIndex.PythonPaths,
		Path:        evt.GetFilename(),
		Global:      d.python.ResourceManager,
		Local:       localGraph,
	}

	// Construct resolver
	resolver := pythonanalyzer.NewResolverUsingImporter(importer, pythonanalyzer.Options{
		User:    d.ids.UserID(),
		Machine: d.ids.MachineID(),
		Path:    d.filename,
	})

	pyctx.Importer = importer
	pyctx.Resolver = resolver

	var resolved *pythonanalyzer.ResolvedAST
	err := ctx.WithTimeout(pyctx.ResolverTimeout, func(ctx kitectx.Context) (err error) {
		resolved, err = pyctx.Resolver.ResolveContext(ctx, pyctx.AST, false)
		return
	})
	if err != nil {
		if _, ok := err.(kitectx.ContextExpiredError); ok {
			resolveTimeout.Hit()
		} else {
			resolveTimeout.Miss()
		}
		return
	}

	pyctx.Resolved = resolved

	// Build the buffer index
	var bufferIndex *bufferIndex
	if resolved != nil && resolved.Module != nil && len(pyctx.Buffer) > 0 {
		bufferIndex = newBufferIndex(ctx, resolved, pyctx.Buffer, pyctx.FileName)
	}

	pyctx.BufferIndex = bufferIndex
}
