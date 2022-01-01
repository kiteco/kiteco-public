package editorapi

import (
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Endpoint describes the endpoints that a language must satisfy
// in order to serve the editor API.
// All endpoints return the specified response, a response code, and an error.
type Endpoint interface {
	Language() lang.Language
	ValueReport(ctx kitectx.Context, id string) (*ReportResponse, int, error)
	ValueMembersExt(ctx kitectx.Context, id string, offset, limit int) (*MembersExtResponse, int, error)
	ValueMembers(ctx kitectx.Context, id string, offset, limit int) (*MembersResponse, int, error)
	SymbolReport(ctx kitectx.Context, id string) (*ReportResponse, int, error)
	Search(ctx kitectx.Context, query string, offset, limit int) (*SearchResults, int, error)
}
