package conversion

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	rcKiteTarget     = "https://" + domains.RemoteConfig
	countryISOURL    = rcKiteTarget + "/convcohort/country/iso"
	emailRequiredURL = rcKiteTarget + "/convcohort/country/email-required"
)

// SwapCountryISOURL is exported for testing
func SwapCountryISOURL(tmp string) (undo func()) {
	countryISOURL, tmp = tmp, countryISOURL
	return func() {
		SwapCountryISOURL(tmp)
	}
}

// SwapEmailRequiredURL is exported for testing
func SwapEmailRequiredURL(tmp string) (undo func()) {
	emailRequiredURL, tmp = tmp, emailRequiredURL
	return func() {
		SwapEmailRequiredURL(tmp)
	}
}

// FetchCountryISO ...
func FetchCountryISO(ctx kitectx.Context) (string, error) {
	type isoResponse struct {
		CountryISO string `json:"country_iso"`
	}

	b, err := doPost(ctx, countryISOURL)
	if err != nil {
		return "", err
	}
	var r isoResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}

	return r.CountryISO, nil
}

// FetchEmailRequired ...
func FetchEmailRequired(ctx kitectx.Context) (bool, error) {
	type reqResp struct {
		EmailRequired bool `json:"email_required"`
	}

	b, err := doPost(ctx, emailRequiredURL)
	if err != nil {
		return false, err
	}
	var r reqResp
	if err := json.Unmarshal(b, &r); err != nil {
		return false, err
	}

	return r.EmailRequired, nil
}

func doPost(ctx kitectx.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx.Context(), "POST", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("returned status %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
