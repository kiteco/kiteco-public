package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/lsp/types"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// DidChange handles an LSP textDocument/didChange call
func (h *Handlers) DidChange(params types.DidChangeTextDocumentParams) error {
	filepath, err := filepathFromURI(params.TextDocument.URI)
	if err != nil {
		return err
	}

	changes := params.ContentChanges
	if len(changes) == 0 {
		return nil
	}

	text := changes[len(changes)-1].Text
	h.files[filepath] = text

	event := component.EditorEvent{
		Source:   string(data.JupyterEditor),
		Action:   "edit",
		Filename: filepath,
		Text:     text,
	}
	buf, _ := json.Marshal(event)
	_, err = http.Post(eventURL, contentType, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	return nil
}
