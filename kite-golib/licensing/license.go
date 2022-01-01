package licensing

import (
	"crypto/rsa"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// License defines a single license for a user
// It simplifies to access the properties of the license data.
// Use "Token()" if you need to store or send the license data.
type License struct {
	Claims
	Token string
}

// GetProduct returns the licensed Product,
// or Free if the License is nil
func (l *License) GetProduct() Product {
	if l == nil {
		return Free
	}
	return l.Product
}

// GetPlan returns the type of Plan (subscription type) associated with the license,
// or FreePlan if the License is nil
func (l *License) GetPlan() Plan {
	if l == nil {
		return FreePlan
	}
	return l.Plan
}

// GetProvID returns the ProvID or ""
func (l *License) GetProvID() string {
	if l == nil {
		return ""
	}
	return l.ProvID
}

// IsPlanActive returns true if the license is still in it's subscription period (ie Now().Before(PlanEnd()))
func (l *License) IsPlanActive() bool {
	if l == nil {
		return true
	}
	return time.Now().Before(l.PlanEnd)
}

// IsExpired returns if the license already expired
func (l *License) IsExpired() bool {
	if l == nil {
		return false
	}
	return !time.Now().Before(l.ExpiresAt)
}

// IsPreferableTo checks if we should prefer l over r.
func (l *License) IsPreferableTo(r *License) bool {
	if l.IsExpired() {
		// l is worse, or they're equally bad
		return false
	}
	if r.IsExpired() {
		// l is better
		return true
	}

	lpo := l.Plan.Order()
	rpo := r.Plan.Order()
	if lpo < rpo {
		return true
	} else if lpo > rpo {
		return false
	}

	return l.ExpiresAt.After(r.ExpiresAt)
}

// IsSubscriber dispatches to l.GetPlan()
func (l *License) IsSubscriber() bool {
	return l.GetPlan().IsSubscriber()
}

// IsPaid dispatches to l.GetPlan()
func (l *License) IsPaid() bool {
	return l.GetPlan().IsPaid()
}

// ParseLicense parses a license, and checks the signature if a key is provided.
func ParseLicense(token string, key *rsa.PublicKey) (*License, error) {
	parser := jwt.Parser{
		ValidMethods:         []string{"RS512"},
		SkipClaimsValidation: false,
	}

	var keyFunc jwt.Keyfunc
	if key != nil {
		keyFunc = func(token *jwt.Token) (interface{}, error) {
			return key, nil
		}
	}

	var claims Claims
	// internally calls claims.Valid()
	parsed, err := parser.ParseWithClaims(token, &claims, keyFunc)
	if err != nil {
		// ParseWithClaims always returns ValidationErrorUnverifiable when keyFunc is nil
		if errJWT, ok := err.(*jwt.ValidationError); keyFunc != nil || !ok || errJWT.Errors != jwt.ValidationErrorUnverifiable {
			return nil, err
		}
	}

	if key != nil {
		if !parsed.Valid {
			return nil, errors.Errorf("token is invalid")
		}
		if err := parsed.Claims.Valid(); err != nil {
			return nil, errors.Errorf("claims are invalid: %v", err)
		}
	}

	return &License{Claims: claims, Token: token}, nil
}
