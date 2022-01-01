package javascript

import (
	"errors"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
)

var errNotImplemented = errors.New("endpoint not implemented")

// Endpoint implements editorapi.Endpoint.
type Endpoint struct {
}

// NewEndpoint creates a new endpoint instance
func NewEndpoint() *Endpoint {
	return &Endpoint{}
}

// Language implements editorapi.Endpoint.
func (e *Endpoint) Language() lang.Language {
	return lang.JavaScript
}

// ValueReport implements editorapi.Endpoint.
func (e *Endpoint) ValueReport(id string) (*editorapi.ReportResponse, int, error) {
	return nil, http.StatusNotImplemented, errNotImplemented
}

// ValueMembers implements editorapi.Endpoint.
func (e *Endpoint) ValueMembers(id string, offset, limit int) (*editorapi.MembersResponse, int, error) {
	return nil, http.StatusNotImplemented, errNotImplemented
}

// ValueLinks implements editorapi.Endpoint.
func (e *Endpoint) ValueLinks(id string, offset, limit int) (*editorapi.LinksResponse, int, error) {
	return nil, http.StatusNotImplemented, errNotImplemented
}

// SymbolReport implements editorapi.Endpoint.
func (e *Endpoint) SymbolReport(id string) (*editorapi.ReportResponse, int, error) {
	return nil, http.StatusNotImplemented, errNotImplemented
}

// Search implements editorapi.Endpoint.
func (e *Endpoint) Search(query string, offset, limit int) (*editorapi.SearchResults, int, error) {
	return nil, http.StatusNotImplemented, errNotImplemented
}
