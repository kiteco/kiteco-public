package python

import (
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// EditorEndpoint provides access to global and local index via HTTP.
// It is meant to be used with a single editor instance i.e. a unique
// combination of user ID and machine ID.
// Implements the editorapi.Endpoint interface.
type EditorEndpoint struct {
	services *Services
	editor   *editorServices
	local    localcode.Context
}

// NewEditorEndpoint creates a new endpoint instance
func NewEditorEndpoint(services *Services, local localcode.Context) *EditorEndpoint {
	return &EditorEndpoint{
		services: services,
		editor:   newEditorServices(services),
		local:    local,
	}
}

// Language implements editorapi.Endpoint.
func (e *EditorEndpoint) Language() lang.Language {
	return lang.Python
}

// ValueReport implements editorapi.Endpoint.
func (e *EditorEndpoint) ValueReport(ctx kitectx.Context, id string) (*editorapi.ReportResponse, int, error) {
	ctx.CheckAbort()

	valueCounter.Add(1)
	defer valueDuration.DeferRecord(time.Now())
	resp, code, err := e.valueReport(ctx, id)
	valueStatusCode.HitAndAdd(strconv.Itoa(code))
	return resp, code, err
}

// valueReport renders a value report response.
func (e *EditorEndpoint) valueReport(ctx kitectx.Context, id string) (*editorapi.ReportResponse, int, error) {
	vb, code, err := e.locateValue(ctx, id)
	if err != nil {
		return nil, code, err
	}
	return &editorapi.ReportResponse{
		Language: lang.Python.Name(),
		Value:    e.editor.renderValueExt(ctx, vb.chooseOne(ctx)),
		Report:   e.editor.renderValueReport(ctx, vb.chooseOne(ctx)),
	}, http.StatusOK, nil
}

// ValueMembers implements editorapi.Endpoint
func (e *EditorEndpoint) ValueMembers(ctx kitectx.Context, id string, offset, limit int) (*editorapi.MembersResponse, int, error) {
	ctx.CheckAbort()

	membersCounter.Add(1)
	defer membersDuration.DeferRecord(time.Now())
	resp, code, err := e.valueMembers(ctx, id, offset, limit)
	membersStatusCode.HitAndAdd(strconv.Itoa(code))
	return resp, code, err
}

// valueMembers renders a value members response
func (e *EditorEndpoint) valueMembers(ctx kitectx.Context, id string, offset, limit int) (*editorapi.MembersResponse, int, error) {
	return e.membersResponse(ctx, id, offset, limit)
}

// ValueMembersExt implements editorapi.Endpoint.
func (e *EditorEndpoint) ValueMembersExt(ctx kitectx.Context, id string, offset, limit int) (*editorapi.MembersExtResponse, int, error) {
	ctx.CheckAbort()

	membersCounter.Add(1)
	defer membersDuration.DeferRecord(time.Now())
	resp, code, err := e.valueMembersExt(ctx, id, offset, limit)
	membersStatusCode.HitAndAdd(strconv.Itoa(code))
	return resp, code, err
}

// valueMembersExt renders a value members response.
func (e *EditorEndpoint) valueMembersExt(ctx kitectx.Context, id string, offset, limit int) (*editorapi.MembersExtResponse, int, error) {
	return e.membersExtResponse(ctx, id, offset, limit)
}

// SymbolReport implements editorapi.Endpoint.
func (e *EditorEndpoint) SymbolReport(ctx kitectx.Context, id string) (*editorapi.ReportResponse, int, error) {
	ctx.CheckAbort()

	symbolCounter.Add(1)
	defer symbolDuration.DeferRecord(time.Now())
	resp, code, err := e.symbolReport(ctx, id)
	symbolStatusCode.HitAndAdd(strconv.Itoa(code))
	return resp, code, err
}

// symbolReport renders a symbol report response.
func (e *EditorEndpoint) symbolReport(ctx kitectx.Context, id string) (*editorapi.ReportResponse, int, error) {
	return e.symbolReportResponse(ctx, id)
}

// Search implements editorapi.Endpoint.
func (e *EditorEndpoint) Search(ctx kitectx.Context, query string, offset, limit int) (*editorapi.SearchResults, int, error) {
	ctx.CheckAbort()

	resp := &editorapi.SearchResults{
		Language: lang.Python.Name(),
	}

	// InvertedIndex is nil in kite local. Return empty search results and set
	// DataUnavailable to true to distinguish from actual empty search results
	if e.services.InvertedIndex == nil {
		resp.DataUnavailable = true
		return resp, http.StatusOK, nil
	}

	results := e.services.InvertedIndex.QueryCompletionLimit(query, offset+limit)

	var i int
	for _, r := range results {
		id := r.Ident
		if len(resp.Results) >= limit {
			break
		}
		if vb, _, err := e.locateValue(ctx, id); err == nil {
			if i >= offset {
				resp.Results = append(resp.Results, editorapi.SearchResult{
					Type:   "value",
					Result: e.editor.renderValue(ctx, vb.chooseOne(ctx)),
				})
			}
			i++
		}
	}
	resp.Start = offset
	resp.End = offset + len(resp.Results)
	resp.Total = len(results)
	return resp, http.StatusOK, nil
}

// LocalSearch is similar to Search but applies to the user's local codebase.
// It is temporarily kept here to keep Search backwards compatible.
// TODO(naman) unused: rm unless we decide to turn local code search back on
func (e *EditorEndpoint) LocalSearch(ctx kitectx.Context, query string, offset, limit int) (*editorapi.SearchResults, int, error) {
	ctx.CheckAbort()
	obj, err := e.local.AnyArtifact()
	if err != nil {
		return nil, http.StatusServiceUnavailable, err
	}
	idx, ok := obj.(*pythonlocal.SymbolIndex)
	if !ok || idx == nil || idx.InvertedIndex == nil {
		return nil, http.StatusServiceUnavailable, errors.Errorf("no index available")
	}

	resp := &editorapi.SearchResults{
		Language: lang.Python.Name(),
	}
	results := idx.InvertedIndex.QueryCompletionLimit(query, offset+limit)

	var i int
	for _, r := range results {
		id := r.Ident
		if len(resp.Results) >= limit {
			break
		}
		if vb, _, err := e.locateValue(ctx, id); err == nil {
			if i >= offset {
				resp.Results = append(resp.Results, editorapi.SearchResult{
					Type:   "value",
					Result: e.editor.renderValue(ctx, vb.chooseOne(ctx)),
				})
			}
			i++
		}
	}
	resp.Start = offset
	resp.End = offset + len(resp.Results)
	resp.Total = len(results)
	return resp, http.StatusOK, nil
}

//
// Responses
//

func (e *EditorEndpoint) symbolReportResponse(ctx kitectx.Context, id string) (*editorapi.ReportResponse, int, error) {
	sb, code, err := e.locateSymbol(ctx, id)
	if err != nil {
		return nil, code, err
	}

	resp := &editorapi.ReportResponse{
		Language: lang.Python.Name(),
		Symbol:   e.editor.renderMemberSymbolExt(ctx, sb),
		Report:   e.editor.renderSymbolReport(ctx, sb.chooseOne(ctx)),
	}

	if ext, ok := sb.val.(pythontype.External); ok {
		sym := ext.Symbol()
		path := sym.Canonical().Path()
		if e.services.SEOData != nil {
			if seoPath := e.services.SEOData.CanonicalLinkPath(sym); !seoPath.Empty() {
				path = seoPath
				resp.LinkCanonical = editorapi.NewID(lang.Python, pythonenv.LocatorForAddress(pythontype.Address{Path: path})).String()
			}
		}
		if e.services.Answers != nil {
			resp.AnswersLinks = e.services.Answers.Links[path.Hash]
		}
	}

	return resp, http.StatusOK, nil
}

func (e *EditorEndpoint) membersResponse(ctx kitectx.Context, id string, off, num int) (*editorapi.MembersResponse, int, error) {
	vb, code, err := e.locateValue(ctx, id)
	if err != nil {
		return nil, code, err
	}

	msbs, total := e.editor.memberSymbols(ctx, vb.chooseOne(ctx), off, num)

	var members []*editorapi.Symbol
	for _, msb := range msbs {
		members = append(members, e.editor.renderMemberSymbol(ctx, msb))
	}
	return &editorapi.MembersResponse{
		Language: lang.Python.Name(),
		Total:    total,
		Start:    off,
		End:      off + len(members),
		Members:  members,
	}, http.StatusOK, nil
}

func (e *EditorEndpoint) membersExtResponse(ctx kitectx.Context, id string, off, num int) (*editorapi.MembersExtResponse, int, error) {
	vb, code, err := e.locateValue(ctx, id)
	if err != nil {
		return nil, code, err
	}

	msbs, total := e.editor.memberSymbols(ctx, vb.chooseOne(ctx), off, num)

	var members []*editorapi.SymbolExt
	for _, msb := range msbs {
		members = append(members, e.editor.renderMemberSymbolExt(ctx, msb))
	}
	return &editorapi.MembersExtResponse{
		Language: lang.Python.Name(),
		Total:    total,
		Start:    off,
		End:      off + len(members),
		Members:  members,
	}, http.StatusOK, nil
}

//
// Value and symbol lookup
//

func makeExternal(graph pythonresource.Manager, path pythonimports.DottedPath) (pythontype.Value, error) {
	if path.Empty() {
		return pythontype.ExternalRoot{Graph: graph}, nil
	}

	sym, err := graph.PathSymbol(path)
	if err != nil {
		return nil, err
	}
	return pythontype.NewExternal(sym, graph), nil
}

func (e *EditorEndpoint) locateValue(ctx kitectx.Context, id string) (valueBundle, int, error) {
	ctx.CheckAbort()

	addr, attr, err := pythonenv.ParseLocator(id)
	if err != nil {
		return valueBundle{}, http.StatusBadRequest, err
	}

	if attr != "" { // compatibility with symbol locators
		addr.Path = addr.Path.WithTail(attr)
	}

	var val pythontype.Value
	var idx *pythonlocal.SymbolIndex

	if addr.File != "" {
		// local
		if e.local == nil {
			return valueBundle{}, http.StatusServiceUnavailable, errors.Errorf("user info not set")
		}

		var ok bool
		idx, ok = e.artifactForFile(addr.File)
		if ok {
			val, err = idx.FindValue(ctx, addr.File, addr.Path.Parts)
			ok = err == nil
		}

		// fall back to buffer index
		if !ok {
			fd, err := e.local.LatestFileDriver(addr.File)

			ok = err == nil
			if ok {
				if unifiedDriver, isUnified := fd.(*UnifiedDriver); isUnified {
					pyctx := unifiedDriver.Context()

					ok = pyctx != nil
					if ok {
						bi := pyctx.BufferIndex
						val, err = bi.FindValue(ctx, addr.File, addr.Path.Parts)

						ok = err == nil
					}
				}
			}
		}

		if !ok {
			return valueBundle{}, http.StatusServiceUnavailable, errors.Errorf("error getting local symbolindex")
		}
	} else {
		// global
		val, err = makeExternal(e.services.ResourceManager, addr.Path)
		if err != nil {
			return valueBundle{}, http.StatusNotFound, errors.Errorf("could not find symbol")
		}
		idx, _ = e.anyArtifact()
	}

	return newValueBundle(ctx, val, indexBundle{
		graph: e.services.ResourceManager,
		idx:   idx,
	}), http.StatusOK, nil
}

func (e *EditorEndpoint) locateSymbol(ctx kitectx.Context, id string) (symbolBundle, int, error) {
	ctx.CheckAbort()

	addr, attr, err := pythonenv.ParseLocator(id)
	if err != nil {
		return symbolBundle{}, http.StatusBadRequest, err
	}

	origFile := addr.File
	if attr == "" {
		if addr.Path.Empty() {
			if origFile == "" {
				return symbolBundle{}, http.StatusNotFound, errors.Errorf("invalid empty locator cannot correspond to symbol")
			}
			attr = strings.TrimSuffix(path.Base(origFile), ".py")
			addr = pythontype.Address{
				User:    addr.User,
				Machine: addr.Machine,
				File:    path.Dir(origFile),
			}
		} else {
			attr = addr.Path.Last()
			addr.Path = addr.Path.Predecessor()
		}
	}

	var nsVal, attrVal pythontype.Value
	var idx *pythonlocal.SymbolIndex

	if origFile != "" {
		// local
		if e.local == nil {
			return symbolBundle{}, http.StatusServiceUnavailable, errors.Errorf("user info not set")
		}

		var ok bool
		// use the original file path here, since the parent directory may not be lookup-able
		// if it doesn't contain an __init__.py, in the case of `site-packages/pkg` or namespace packages.
		idx, ok = e.artifactForFile(origFile)
		if ok {
			nsVal, attrVal, err = idx.FindSymbol(ctx, addr.File, addr.Path.Parts, attr)
			ok = err == nil
		}

		// fall back to buffer index of the latest file driver, if available
		if !ok {
			fd, err := e.local.LatestFileDriver(origFile)

			ok = err == nil
			if ok {
				if unifiedDriver, isUnified := fd.(*UnifiedDriver); isUnified {
					pyctx := unifiedDriver.Context()

					ok = pyctx != nil
					if ok {
						bi := pyctx.BufferIndex
						nsVal, attrVal, err = bi.FindSymbol(ctx, addr.File, addr.Path.Parts, attr)

						ok = err == nil
					}
				}
			}
		}

		if !ok {
			return symbolBundle{}, http.StatusServiceUnavailable, errors.Errorf("error getting local code index for file %s", origFile)
		}
	} else {
		// global
		nsVal, err = makeExternal(e.services.ResourceManager, addr.Path)
		if nsVal == nil || err != nil {
			return symbolBundle{}, http.StatusNotFound, errors.Errorf("could not find namespace for symbol")
		}

		attrVal, err = makeExternal(e.services.ResourceManager, addr.Path.WithTail(attr))
		if attrVal == nil || err != nil {
			return symbolBundle{}, http.StatusNotFound, errors.Errorf("could not find symbol")
		}
		idx, _ = e.anyArtifact()
	}

	return newValueBundle(ctx, nsVal, indexBundle{
		idx:   idx,
		graph: e.editor.services.ResourceManager,
	}).memberSymbol(ctx, attrVal, attr), http.StatusOK, nil
}

func (e *EditorEndpoint) anyArtifact() (*pythonlocal.SymbolIndex, bool) {
	if e.local == nil {
		return nil, false
	}
	if obj, err := e.local.AnyArtifact(); err == nil {
		if idx, ok := obj.(*pythonlocal.SymbolIndex); ok {
			return idx, true
		}
	}
	return nil, false
}

func (e *EditorEndpoint) artifactForFile(path string) (*pythonlocal.SymbolIndex, bool) {
	if e.local == nil {
		return nil, false
	}
	if obj, err := e.local.ArtifactForFile(path); err == nil {
		if idx, ok := obj.(*pythonlocal.SymbolIndex); ok {
			return idx, true
		}
	}
	if !strings.HasSuffix(path, ".py") {
		return e.artifactForFile(path + ".py")
	}
	return nil, false
}
