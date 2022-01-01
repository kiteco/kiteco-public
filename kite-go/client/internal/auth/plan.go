package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/community/account"
)

// ProDefaultPlan is the static answer returned by handlePlan - Deprecated
// Deprecated
// TODO Remove this code when handlePlan endpoints will be removed
var ProDefaultPlan = account.PlanResponse{
	Name:                "pro",
	ActiveSubscription:  "pro",
	Status:              "active",
	Features:            account.FeatureSet{},
	TrialDaysRemaining:  0,
	StartedKiteProTrial: true,
	MaxReferralCredits:  10,
}

// Deprecated
func (c *Client) handlePlan(w http.ResponseWriter, r *http.Request) {
	buf, err := json.Marshal(&ProDefaultPlan)
	if err != nil {
		err = fmt.Errorf("error serializing plan: %v", err)
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}
