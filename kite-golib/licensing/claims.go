package licensing

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Version enumerates the Kite claims versions
type Version string

// Versions
const (
	V1 Version = "1.0"
)

// Claims defines the attributes of a Kite license (encoded as JWT claims)
type Claims struct {
	// Version is used to track the version of the backend that issued the claim
	Version  Version   `json:"ver,omitempty"`
	IssuedAt time.Time `json:"iss_t,omitempty"`

	// ProvID in conjunction with Plan, identifies how the license came to be ("provenance").
	ProvID string `json:"prov_id"`
	Plan   Plan   `json:"plan"`

	UserID    string    `json:"uid,omitempty"`
	InstallID string    `json:"iid,omitempty"`
	ExpiresAt time.Time `json:"exp_t"`
	Product   Product   `json:"prod"`
	PlanEnd   time.Time `json:"plan_end"`
}

// Valid implements jwt.Claims.
// It does not check that the license has not expired.
func (c Claims) Valid() error {
	// assert version == 1.0
	if c.Version != "1.0" {
		return errors.Errorf("unsupported version")
	}

	if c.UserID == "" && c.InstallID == "" {
		return errors.Errorf("both uid and iid are empty")
	}

	// assert iss_t > 0
	if c.IssuedAt.IsZero() {
		return errors.Errorf("iss_t is zero")
	}

	// assert iss_t <= now
	if time.Now().Before(c.IssuedAt) {
		return errors.Errorf("iss_t is in the future")
	}

	// assert iss_t <= plan_end
	if c.PlanEnd.Before(c.IssuedAt) {
		return errors.Errorf("plan_end is before iss_t")
	}
	// assert plan_end <= exp_t
	if c.ExpiresAt.Before(c.PlanEnd) {
		return errors.Errorf("exp_t is before plan_end")
	}

	if !c.Plan.IsValid() {
		return errors.Errorf("invalid plan")
	}
	if c.Product != c.Plan.Product() {
		return errors.Errorf("product does not match plan")
	}
	if c.Product == Free {
		return errors.Errorf("cannot issue license for Kite Free product")
	}
	return nil
}
