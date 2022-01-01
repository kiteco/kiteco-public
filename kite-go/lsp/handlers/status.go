package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/lsp/types"
	"github.com/kiteco/kiteco/kite-golib/enginestatus"
)

// Status gets the state of the Kite Engine for the given file.
func (h *Handlers) Status(params types.KiteStatusParams) (enginestatus.Response, error) {
	filepath, err := filepathFromURI(params.URI)
	if err != nil {
		return enginestatus.Response{}, err
	}
	statusPath, err := buildURL(statusURL, map[string]string{"filename": filepath})
	if err != nil {
		return enginestatus.Response{}, err
	}
	res, err := http.Get(statusPath)
	if err != nil {
		return enginestatus.Response{}, err
	}

	kiteResponse := enginestatus.Response{}
	err = json.NewDecoder(res.Body).Decode(&kiteResponse)
	if err != nil {
		return enginestatus.Response{}, err
	}
	return kiteResponse, nil
}
