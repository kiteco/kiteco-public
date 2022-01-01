package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/client/component"

	"github.com/kiteco/kiteco/kite-go/lsp/types"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

type completionRequest struct {
	Request  data.APIRequest
	Text     string         `json:"text"`
	Position data.Selection `json:"position"`
}

// Completion handles an LSP textDocument/completion call and returns a response
func (h *Handlers) Completion(params types.CompletionParams) (types.CompletionList, error) {
	filepath, err := filepathFromURI(params.TextDocument.URI)
	if err != nil {
		return types.CompletionList{}, err
	}

	text := h.files[filepath]
	off, err := utf8OffFromPos(text, params.Position)
	if err != nil {
		return types.CompletionList{}, err
	}

	// editor event
	event := component.EditorEvent{
		Source:     string(data.JupyterEditor),
		Action:     "selection",
		Filename:   filepath,
		Text:       text,
		Selections: []*component.Selection{component.NewSelection(data.Cursor(off), stringindex.UTF8)},
	}
	editBuf, _ := json.Marshal(event)
	_, err = http.Post(eventURL, contentType, bytes.NewBuffer(editBuf))
	if err != nil {
		// We should still try and fetch completions, so just log this error
		log.Printf("%q\n", err)
	}

	// completion request
	r := data.APIRequest{
		UMF: data.UMF{Filename: filepath},
		APIOptions: data.APIOptions{
			Editor:     data.JupyterEditor,
			NoSnippets: true,
			Encoding:   stringindex.UTF8,
		},
		SelectedBuffer: data.NewBuffer(text).Select(data.Cursor(off)),
	}
	buf, err := json.Marshal(r)
	if err != nil {
		return types.CompletionList{}, err
	}
	resp, err := http.Post(completeURL, contentType, bytes.NewBuffer(buf))
	if err != nil {
		return types.CompletionList{}, err
	}

	kiteResponse := data.NewAPIResponse(r)
	err = json.NewDecoder(resp.Body).Decode(&kiteResponse)
	if err != nil {
		return types.CompletionList{}, err
	}

	var items []types.CompletionItem
	for _, c := range kiteResponse.Completions {
		items = append(items, h.translateCompletion(c.RCompletion, text))
	}

	return types.CompletionList{Items: items}, nil
}

// translateCompletion turns a Kite Completion into an LSP Completion
func (h *Handlers) translateCompletion(c data.RCompletion, text string) types.CompletionItem {
	start, err := posFromUTF8Off(text, c.Replace.Begin)
	if err != nil {
		panic(err)
	}
	end, err := posFromUTF8Off(text, c.Replace.End)
	if err != nil {
		panic(err)
	}

	item := types.CompletionItem{
		Label:            c.Display,
		InsertTextFormat: types.PlainTextTextFormat,
		InsertText:       c.Snippet.Text,
		TextEdit: &types.TextEdit{
			Range: types.Range{
				Start: start,
				End:   end,
			},
			NewText: c.Snippet.Text,
		},
		Documentation: addDocsBranding(c.Docs.Text),
	}

	// LSP clients must send {kiteTypesEnabled: true} to receive a hint string.
	opts, ok := h.Options.(map[string]interface{})
	kte, ok := opts[kiteTypesEnabled].(bool)

	if c.Hint != "" {
		if ok && kte {
			item.KiteHint = c.Hint
		} else {
			item.Kind = kindForHint(c.Hint)
		}

	}

	return item
}

func addDocsBranding(s string) string {
	if s == "" {
		return "[Kite]"
	}
	return "[Kite]\n\n" + s
}
