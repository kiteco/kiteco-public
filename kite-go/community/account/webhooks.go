package account

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/mixpanel"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const (
	secret = "XXXXXXX"
)

func validateSecretKey(s string) error {
	if s != secret {
		return fmt.Errorf("invalid secret key provided")
	}
	return nil
}

// HandleActivatedUser is a webhook that receives a signal from customer.io
// when a user activates
func (s *Server) HandleActivatedUser(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.HandleActivatedUser:"

	var request struct {
		Secret   string `json:"secret"`
		Email    string `json:"email"`
		Language string `json:"language"`
	}

	if ed := unmarshalBody(lp, r, &request); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	// be sure to send the secret key over
	if err := validateSecretKey(request.Secret); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Verify that this user exists in our db
	user, err := s.app.Users.FindByEmail(request.Email)
	if err != nil {
		webutils.ErrorResponse(w, r, err, community.UserErrorMap)
		return
	}

	// request info from mixpanel
	users, err := s.mixpanelManager.RequestUserInfo(request.Email)
	if err != nil {
		rollbar.Error(err)
	}

	// need to filter out for the correct user
	var selectedUser mixpanel.User
	for _, u := range users {
		if u.DistinctID == user.IDString() {
			selectedUser = u
		}
	}

	delighted := createDelightedPeople(user.Email, request.Language, selectedUser)
	if selectedUser.DistinctID == "" {
		rollbar.Error(fmt.Errorf("unable to find user in mixpanel jql query results"))
	} else {
		// send to segment -> to customer.io
		community.AddTraits(user.IDString(), delighted.Properties)
	}

	// finally, send off to delighted
	err = s.delightedManager.sendDelightedSurvey(delighted)
	if err != nil {
		rollbar.Error(err)
		http.Error(w, err.Error(), http.StatusNotFound)
	}
}

// HandleDelightedEvent is a webhook that receives a signal from delighted
// when a user submits a survey result
func (s *Server) HandleDelightedEvent(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.HandleDelightedEvent:"

	ok, err := s.delightedManager.verifyWebhook(r)
	if err != nil {
		rollbar.Error(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if !ok {
		err = fmt.Errorf("could not verify delighted webhook")
		rollbar.Error(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var event DelightedEvent
	if ed := unmarshalBody(lp, r, &event); ed.HTTPError() {
		rollbar.Error(errors.New("could not unmarshal delighted event"))
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	// Verify that this user exists in our db
	user, err := s.app.Users.FindByEmail(event.EventData.Person.Email)
	if err != nil {
		rollbar.Error(errors.New("could not find delighted person email"))
		webutils.ErrorResponse(w, r, err, community.UserErrorMap)
		return
	}

	// Send off to customer.io
	properties := map[string]interface{}{
		"last_nps_score":           event.EventData.Score,
		"last_nps_submission_time": time.Now().Unix(),
	}

	community.AddTraits(user.IDString(), properties)
	community.TrackNPS(user.IDString(), event.EventData.Score, event.EventData.Comment)
}
