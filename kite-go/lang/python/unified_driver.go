package python

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-go/diff"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonhelpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/status"
)

const (
	// fraction of completions to sample and send to Segment
	completionsSampleFraction = 0.1
)

// UnifiedDriver is the entrypoint for python file events (implements core.FileDriver)
type UnifiedDriver struct {
	debug       bool
	diagnostics io.Writer

	file         core.FileDriver
	completions  *response.EditorCompletions
	prefetched   []*response.EditorCompletions
	indexPresent *response.LocalIndexPresent

	python      *Services
	lastEvent   *event.Event
	lastContext *Context

	scanOpts  pythonscanner.Options
	incrLexer *pythonscanner.Incremental

	localContext localcode.Context

	// mediates between HTTP endpoints, which are read-only, and HandleEvent, which
	// modifies the state of this driver
	lock sync.RWMutex

	// Name of the file currently opened by the user.
	filename string
	ids      userids.IDs

	// passive search id for electron app
	autosearchID string

	editor    *editorServices
	kiteLocal bool
}

// NewUnifiedDriver creates a new Python driver.
func NewUnifiedDriver(ids userids.IDs, filename string, python *Services, local localcode.Context, kiteLocal bool) *UnifiedDriver {
	scanOpts := pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	}
	driver := &UnifiedDriver{
		file:         diff.NewBufferDriver(),
		python:       python,
		scanOpts:     scanOpts,
		localContext: local,
		ids:          ids,
		filename:     filename,
		editor:       newEditorServices(python),
		kiteLocal:    kiteLocal,
		incrLexer:    pythonscanner.NewIncrementalFromBuffer(nil, scanOpts),
	}

	return driver
}

// InitSetDebug sets debug. It is not thread-safe.
func (d *UnifiedDriver) InitSetDebug() {
	d.debug = true
}

// Context returns the most recent Context
func (d *UnifiedDriver) Context() *Context {
	d.lock.RLock()
	defer d.lock.RUnlock()
	if d.lastContext == nil {
		return nil
	}
	ctx := *d.lastContext
	return &ctx
}

// Resolved ast for the file, this is read only
func (d *UnifiedDriver) Resolved() (*pythonanalyzer.ResolvedAST, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	if d.lastContext == nil {
		return nil, fmt.Errorf("last context is nil")
	}
	return d.lastContext.Resolved, nil
}

// Callee calls GetCallee on the current context, checks for errors, and returns any potential error and http status code
func (d *UnifiedDriver) Callee(ctx kitectx.Context, cursor int64) (CalleeResult, int, error) {
	ctx.CheckAbort()

	// we need to guard access to d.lastContext
	// since we cannot guarantee that d.HandleEvent
	// is not being executed in another go routine.
	inputs, ok := func() (CalleeInputs, bool) {
		d.lock.RLock()
		defer d.lock.RUnlock()
		if d.lastContext == nil {
			// this can happen because we do not guarantee
			// that a unified driver has processed its first
			// event before Callee requests have been sent.
			// SEE: https://github.com/kiteco/kiteco/blob/master/kite-go/client/internal/kitelocal/file_processor.go
			return CalleeInputs{}, false
		}

		return NewCalleeInputs(d.lastContext, cursor, d.python), true
	}()

	if !ok {
		return CalleeResult{Failure: pythontracking.NoContextFailure}, http.StatusServiceUnavailable, errors.New("driver.lastContext is nil")
	}

	result := GetCallee(ctx, inputs)

	if result.Response == nil {
		httpStatus := http.StatusInternalServerError
		errorMsg := "unknown callee failure"
		switch result.Failure {
		case pythontracking.NoCallExprFailure:
			httpStatus = http.StatusBadRequest
			errorMsg = "call expression not found"
			missingCalleeReason.HitAndAdd("Call expression not found")
		case pythontracking.OutsideParensFailure:
			httpStatus = http.StatusBadRequest
			errorMsg = "call expression not found"
			missingCalleeReason.HitAndAdd("Cursor in call expression but outside parens")
		case pythontracking.NilRefFailure:
			httpStatus = http.StatusNotFound
			errorMsg = "expression has a null reference"
			missingCalleeReason.HitAndAdd("Null reference")
		case pythontracking.UnresolvedValueFailure:
			httpStatus = http.StatusNotFound
			errorMsg = "expression has an unresolved value"
			if d.lastContext.LocalIndex == nil {
				missingCalleeReason.HitAndAdd("Unresolved value (no index)")
			} else {
				missingCalleeReason.HitAndAdd("Unresolved value")
			}
		case pythontracking.InvalidKindFailure:
			httpStatus = http.StatusNotFound
			errorMsg = "invalid callee kind"
			missingCalleeReason.HitAndAdd("Invalid callee kind")
		case pythontracking.ValTranslateFailure:
			errorMsg = "translated value is nil"
			missingCalleeReason.HitAndAdd("Nil translated value")
		}
		return result, httpStatus, errors.New(errorMsg)
	}

	return result, http.StatusOK, nil
}

// HandleEvent implements core.Driver
func (d *UnifiedDriver) HandleEvent(ctx kitectx.Context, evt *event.Event) string {
	ctx.CheckAbort()

	d.lock.Lock()
	defer d.lock.Unlock()

	start := time.Now()
	defer func() {
		handleEventDuration.RecordDuration(time.Since(start))
		ctx.Logger.Durations.Record("python/UnifiedDriver.HandleEvent", time.Since(start))
	}()

	// copy last event
	d.lastEvent = evt

	fileStart := time.Now()
	state := d.file.HandleEvent(ctx, evt)
	bufferDuration.RecordDuration(time.Since(fileStart))

	// compute the context
	contextStart := time.Now()
	pyctx := d.parse(ctx, evt)
	contextDuration.RecordDuration(time.Since(contextStart))
	ctx.Logger.Durations.Record("python/UnifiedDriver.HandleEvent (d.parse)", time.Since(contextStart))
	if pyctx == nil {
		return state
	}

	// run this driver
	handleStart := time.Now()
	d.handle(ctx, evt, pyctx)
	handleDuration.RecordDuration(time.Since(handleStart))
	ctx.Logger.Durations.Record("python/UnifiedDriver.HandleEvent (d.handle)", time.Since(handleStart))
	return state
}

const maxIndicies = 5

// CollectOutput impelements core.Driver
func (d *UnifiedDriver) CollectOutput() []interface{} {
	// Note: this function holds a read-only lock, so it must not modify the driver state
	d.lock.RLock()
	defer d.lock.RUnlock()

	var ret []interface{}
	if d.completions != nil {
		ret = append(ret, d.completions)
	}
	if d.prefetched != nil {
		ret = append(ret, d.prefetched)
	}
	if d.autosearchID != "" {
		ret = append(ret, &response.Autosearch{
			AutosearchID: d.autosearchID,
		})
	}
	if d.indexPresent != nil {
		ret = append(ret, d.indexPresent)
	}
	if d.localContext != nil {
		ret = append(ret, d.localContext.StatusResponse(maxIndicies))
	}

	return ret
}

// Bytes implements core.FileDriver.
func (d *UnifiedDriver) Bytes() []byte {
	return d.file.Bytes()
}

// Cursor implements core.FileDriver.
func (d *UnifiedDriver) Cursor() int64 {
	return d.file.Cursor()
}

// SetContents implements core.FileDriver.
func (d *UnifiedDriver) SetContents(buf []byte) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	d.file.SetContents(buf)
}

// ResendText implements core.FileDriver.
func (d *UnifiedDriver) ResendText() bool {
	return d.file.ResendText()
}

// parse the contents of the file and trigger appropriate handler functions
func (d *UnifiedDriver) parse(ctx kitectx.Context, evt *event.Event) *Context {
	ctx.CheckAbort()

	// Always attempt to get a local index
	var localIndex *pythonlocal.SymbolIndex
	var artifactError error
	if d.localContext != nil {
		obj, err := d.localContext.RequestForFile(evt.GetFilename(), map[string]interface{}{
			"evt_source": evt.GetSource(),
		})
		artifactError = err
		if err == nil {
			localIndex = obj.(*pythonlocal.SymbolIndex)
			d.printf("got local index")
		}
	}

	d.indexPresent = &response.LocalIndexPresent{
		Present: localIndex != nil,
	}

	// Always update the last context with the latest index
	if d.lastContext != nil {
		d.lastContext.LocalIndex = localIndex
	}

	// If we are responding to a selection event, we can avoid fully rebuilding
	// the context (parse, resolve, etc).
	if d.lastContext != nil && evt.GetAction() == "selection" {
		d.printf("reusing previous context")
		reuseContextRatio.Hit()
		d.updateContextWithSelection(ctx, d.lastContext, evt, d.Cursor())
		return d.lastContext
	}

	d.printf("rebuilding context")
	reuseContextRatio.Miss()

	// update incremental lexer, need to do this with every event
	// to make sure any updates to the text are tracked by the incremental lexer
	// e.g a focus event sends new version of full text, which may have changes
	// so need to relex or update.
	start := time.Now()
	var relex bool
	for _, diff := range evt.GetDiffs() {
		if trigger := d.incrLexer.Update(incrementFromDiff(diff)); trigger != nil {
			relex = true
			break
		}
	}
	ctx.Logger.Durations.Record("python/UnifiedDriver.parse (incrLexer)", time.Since(start))

	// need second clause for cases in which kited sent back
	// full editor text (e.g new file, focus, resend) in event,
	// and incremental needs to relex new buffer. This also
	// acts as a safety measure in case the two buffers ever get out
	// of sync.
	start = time.Now()
	if relex || !bytes.Equal(d.Bytes(), d.incrLexer.Buffer()) {
		d.incrLexer = pythonscanner.NewIncrementalFromBuffer(d.Bytes(), d.scanOpts)
		relexRatio.Hit()
	} else {
		relexRatio.Miss()
	}
	ctx.Logger.Durations.Record("python/UnifiedDriver.parse (NewIncrementalFromBuffer)", time.Since(start))

	start = time.Now()
	defer func() {
		contextDuration.RecordDuration(time.Since(start))
	}()

	// Construct import environment
	importer := pythonstatic.Importer{
		Path:   evt.GetFilename(),
		Global: d.python.ResourceManager,
	}

	if localIndex != nil {
		haveLocalGraph.Hit()
		importer.Local = localIndex.SourceTree
		importer.PythonPaths = localIndex.PythonPaths
	} else {
		haveLocalGraph.Miss()
	}

	// Construct resolver
	resolver := pythonanalyzer.NewResolverUsingImporter(importer, pythonanalyzer.Options{
		User:    d.ids.UserID(),
		Machine: d.ids.MachineID(),
		Path:    d.filename,
	})

	var sel [2]int64
	if s := evt.GetSelections(); len(s) == 1 {
		sel[0], sel[1] = s[0].GetStart(), s[0].GetEnd()
		if sel[0] > sel[1] {
			sel[0], sel[1] = sel[1], sel[0]
		}
	}

	// Construct context
	bufferSizeSample.Record(int64(len(d.Bytes())))

	newContextStart := time.Now()
	pyctx, err := NewContext(ctx, ContextInputs{
		User:            d.ids.UserID(),
		Machine:         d.ids.MachineID(),
		Buffer:          d.Bytes(),
		Cursor:          d.Cursor(),
		Selection:       sel,
		LastAction:      evt.GetAction(),
		FileName:        evt.GetFilename(),
		Importer:        importer,
		Resolver:        resolver,
		IncrLexer:       d.incrLexer,
		LocalIndex:      localIndex,
		D:               d.diagnostics,
		EventSource:     evt.GetSource(),
		ResolverTimeout: defaultResolverTimeout,
		ArtifactError:   artifactError,
	})
	newContextDuration.RecordDuration(time.Since(newContextStart))
	ctx.Logger.Durations.Record("python/UnifiedDriver.parse (NewContext)", time.Since(newContextStart))
	if pyctx == nil || err != nil {
		parseErrorRatio.Hit()
	} else {
		parseErrorRatio.Miss()
	}

	d.lastContext = pyctx

	// the importer is needed later to do top-level import completions
	return pyctx
}

func (d *UnifiedDriver) handle(ctx kitectx.Context, evt *event.Event, pyctx *Context) {
	ctx.CheckAbort()

	// TODO(juan): hack to get electron to work consistently
	d.autosearchID = d.renderAutosearchID(ctx, pyctx)
	d.println("autosearch id:", d.autosearchID)

	d.recordMetrics(ctx, pyctx, evt)
}

func (d *UnifiedDriver) renderAutosearchID(ctx kitectx.Context, pyctx *Context) string {
	ctx.CheckAbort()

	var begin, end int64
	if pyctx.LastAction == "edit" {
		begin, end = pyctx.TrimmedCursor, pyctx.TrimmedCursor
	} else {
		begin, end = pyctx.Selection[0], pyctx.Selection[1]
	}

	d.println("in renderAutosearchID:")
	namePrinter := func(fmtstr string, values ...interface{}) {
		d.printf("  "+fmtstr, values...)
	}

	_, sbs, err := resolveNode(ctx, pythonhelpers.DeepestContainingSelection(ctx, pyctx.AST, begin, end), resolveInputs{
		LocalIndex:  pyctx.LocalIndex,
		BufferIndex: pyctx.BufferIndex,
		Resolved:    pyctx.Resolved,
		Graph:       pyctx.Importer.Global,
		PrintDebug:  namePrinter,
	})

	if err != nil {
		return ""
	}

	// TODO(juan): this should really be a list,
	// SEE: https://github.com/kiteco/kiteco/issues/5082
	return d.editor.renderSymbolID(ctx, sbs[0]).String()
}

func (d *UnifiedDriver) recordMetrics(ctx kitectx.Context, pyctx *Context, evt *event.Event) {
	var matchNodeRatio, resolveRatio *status.Ratio
	var nodeType *status.Breakdown
	switch pyctx.LastAction {
	case "edit":
		matchNodeRatio = editMatchNodeRatio
		resolveRatio = editResolveRatio
		nodeType = editNodeType
	case "selection":
		matchNodeRatio = selectionMatchNodeRatio
		resolveRatio = selectionResolveRatio
		nodeType = selectionNodeType
	default:
		matchNodeRatio = otherMatchNodeRatio
		resolveRatio = otherResolveRatio
		nodeType = otherNodeType
	}

	// Look at the nodes under the cursor and find the deepest NameExpr/AttributeExpr/CallExpr/DottedExpr,
	// as well as the reference it resolves to
	var matchedExpr pythonast.Expr
	var rv pythontype.Value

	for _, node := range pyctx.UnderCursor {
		switch node := node.(type) {
		case *pythonast.CallExpr:
			matchedExpr = node
		case *pythonast.AttributeExpr:
			matchedExpr = node
		case *pythonast.NameExpr:
			matchedExpr = node
		case *pythonast.DottedExpr:
			matchedExpr = node
		}
		if matchedExpr != nil {
			rv = pyctx.Resolved.References[matchedExpr]
		}
	}

	if matchedExpr != nil {
		matchNodeRatio.Hit()
		nodeType.Hit(reflect.TypeOf(matchedExpr).String())
	} else {
		matchNodeRatio.Miss()
	}

	if rv != nil {
		resolveRatio.Hit()
	} else {
		resolveRatio.Miss()
	}
}

// StartDiagnostics implements the core.Diagnoser interface
func (d *UnifiedDriver) StartDiagnostics(w io.Writer) {
	d.diagnostics = w
}

func (d *UnifiedDriver) printf(str string, values ...interface{}) {
	if d.diagnostics != nil {
		fmt.Fprintf(d.diagnostics, str, values...)
		if !strings.HasSuffix(str, "\n") {
			fmt.Fprint(d.diagnostics, "\n")
		}
	}
	if d.debug {
		log.Printf(str, values...)
	}
}

func (d *UnifiedDriver) println(parts ...interface{}) {
	if d.diagnostics != nil {
		fmt.Fprintln(d.diagnostics, parts...)
	}
	if d.debug {
		log.Println(parts...)
	}
}
