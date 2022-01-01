package account

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type slackManager struct {
	token string
}

func newSlackManager(token string) *slackManager {
	return &slackManager{token}
}

func (s *slackManager) Invite(email string) error {
	if s.token == "" {
		return nil
	}

	u, err := url.Parse("https://slack.com/api/users.admin.invite")
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set("token", s.token)
	q.Set("email", email)
	q.Set("channels", "C0ZQG2MHV,C69TRL2BV,C69E5BGJV,C6A4BMQ6R,C0ZQG2MLP,C6ASP24BG,C6A4BK7N1,C6A4BFJLD")
	u.RawQuery = q.Encode()

	req, err := http.Get(u.String())
	if err != nil {
		return fmt.Errorf("slack_failure")
	}

	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("slack_failure")
	}

	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(buf, &response); err != nil {
		return fmt.Errorf("invalid_json")
	}

	if !response.OK {
		return fmt.Errorf(response.Error)
	}

	return nil
}
