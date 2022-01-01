package component

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

// Selection represents a region of text that is selected
type Selection struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`

	Encoding stringindex.OffsetEncoding `json:"encoding"`
}

// EditorEvent represents a modification to the state of an open file in a buffer
type EditorEvent struct {
	Source        string       `json:"source"`
	Action        string       `json:"action"`
	Filename      string       `json:"filename"`
	Text          string       `json:"text"`
	Selections    []*Selection `json:"selections"`
	EditorVersion string       `json:"editor_version,omitempty"`
	PluginVersion string       `json:"plugin_version,omitempty"`

	Timestamp time.Time
}

// NewSelection converts a data.Selection into a component.Selection
func NewSelection(sel data.Selection, enc stringindex.OffsetEncoding) *Selection {
	return &Selection{
		Start:    int64(sel.Begin),
		End:      int64(sel.End),
		Encoding: enc,
	}
}
