package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/community/account"
	cohorts "github.com/kiteco/kiteco/kite-golib/conversion"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

// LicenseInfo extends licensing.LicenseInfo
type LicenseInfo struct {
	licensing.LicenseInfo
	// TrialAvailableDuration is a human-readable string with the available trial's duration.
	// e.g. "4 weeks", "6 weeks", etc. It is not set if no trial is available.
	TrialAvailableDuration *TrialAvailableDuration `json:"trial_available_duration,omitempty"`
}

// TrialAvailableDuration breaks down the trial duration into singular unit and value for rendering.
type TrialAvailableDuration struct {
	Unit  string `json:"unit"`
	Value int    `json:"value"`
}

// Crude time constants for general representation.
const (
	day  = time.Hour * 24
	week = day * 7
)

func (c *Client) getLicenseInfo() LicenseInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// i := LicenseInfo{LicenseInfo: c.licenseStore.LicenseInfo()}
	i := LicenseInfo{LicenseInfo: licensing.LicenseInfo{
		Product: licensing.Pro,
		Plan:    licensing.ProIndefinite,
	}}
	if !i.LicenseInfo.Plan.IsSubscriber() && c.cohort.ConversionCohort() == cohorts.UsagePaywall {
		// If the cohort changes in the middle of a trial to usage-paywall
		// the copilot must be not be informed about the stored license
		i.Product = licensing.Free
		i.Plan = licensing.FreePlan
	}
	if i.TrialAvailable {
		trialdur, err := c.settings.GetDuration(settings.TrialDuration)
		if err != nil || trialdur == 0 {
			i.TrialAvailableDuration = &TrialAvailableDuration{
				Unit:  "week",
				Value: 4,
			}
		} else {
			trialdur = trialdur.Round(week)
			durInWeeks := int(trialdur.Nanoseconds() / week.Nanoseconds())
			i.TrialAvailableDuration = &TrialAvailableDuration{
				Unit:  "week",
				Value: durInWeeks,
			}
		}
	}

	return i
}

func (c *Client) handleLicenseInfo(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("refresh") != "false" {
		c.RefreshLicenses(r.Context())
	}

	info := c.getLicenseInfo()
	buf, err := json.Marshal(info)
	if err != nil {
		err = errors.Errorf("error serializing planV2 info: %v", err)
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (c *Client) getRemoteTokensLocked(ctx context.Context) ([]string, error) {
	dest := "/api/account/licenses"
	queries := url.Values{}
	queries.Add("install-id", c.userIDs.InstallID())
	dest = dest + "?" + queries.Encode()

	resp, err := c.getNoHMACLocked(ctx, dest)
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching %s", dest)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		var licenses account.LicensesResponse
		if err := json.NewDecoder(resp.Body).Decode(&licenses); err != nil {
			return nil, errors.Wrapf(err, "error unmarshalling licenses from %s", dest)
		}
		return licenses.LicenseTokens, nil
	case resp.StatusCode == http.StatusUnauthorized:
		// Only return ErrNotAuthenticated if we get http.StatusUnauthorized
		return nil, ErrNotAuthenticated
	default:
		return nil, errors.Errorf("got response code %d fetching %s", resp.StatusCode, dest)
	}
}
