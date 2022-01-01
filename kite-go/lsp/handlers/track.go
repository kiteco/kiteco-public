package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/lsp/types"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
)

// Track sends a tracking request to kited
func (h *Handlers) Track(params types.KiteTrackParams) error {
	if params.To != "mixpanel" {
		return errors.Errorf("cannot track to %s", params.To)
	}
	trackPath, err := buildURL(trackMixpanelURL, nil)
	if err != nil {
		return err
	}
	trackParams := telemetry.CustomEvent{
		Event: params.Event,
		// Matches key in livemetrics
		Key:   "XXXXXXX",
		Props: params.Props,
	}
	buf, err := json.Marshal(trackParams)
	if err != nil {
		return err
	}
	r := bytes.NewReader(buf)
	http.Post(trackPath, "application/json", r)
	return nil
}
