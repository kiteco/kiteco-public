package legacy

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonhelpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Completion represents a proposed completion for an editing situation.
type Completion struct {
	Identifier string
	// Referent is the value that represents the given completion. This can be nil, since not all completions
	// (for example keyword completions) resolve to a value.
	Referent pythontype.Value
	// If the completions were ranked by scoring, Score holds the score that was computed for the completion. There is
	// no guarantee that the relative rank of the completions is by score.
	Score float64

	Source response.EditorCompletionSource
}

// Inputs contains information used to match a Situation and provide and score completions. All of this information
// should be derived only from the AST, buffer, and cursor.
type Inputs struct {
	AST                pythonast.Node
	Buffer             []byte
	Cursor             int64
	Words              []pythonscanner.Word
	UnderCursor        []pythonast.Node
	TrimmedCursor      int64
	UnderTrimmedCursor []pythonast.Node
	IDs                userids.IDs
}

// Inputs creates Inputs given a buffer, cursor and AST.
func (i CompletionsCallbacks) Inputs(ctx kitectx.Context) Inputs {
	ctx.CheckAbort()

	trimmedCursor := pythonhelpers.NearestNonWhitespace(i.Buffer, i.Cursor, pythonhelpers.IsHSpace)
	return Inputs{
		AST:                i.Resolved.Root,
		Buffer:             i.Buffer,
		Cursor:             i.Cursor,
		Words:              i.Words,
		UnderCursor:        pythonhelpers.NodesUnderCursor(ctx, i.Resolved.Root, i.Cursor),
		TrimmedCursor:      trimmedCursor,
		UnderTrimmedCursor: pythonhelpers.NodesUnderCursor(ctx, i.Resolved.Root, trimmedCursor),
		IDs:                i.IDs,
	}
}
