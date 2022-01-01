package handlers

import (
	"github.com/kiteco/kiteco/kite-go/lsp/types"
)

// DidClose handles an LSP textDocument/didClose call
func (h *Handlers) DidClose(params types.DidCloseTextDocumentParams) error {
	filepath, err := filepathFromURI(params.TextDocument.URI)
	if err != nil {
		return err
	}

	delete(h.files, filepath)
	return nil
}
