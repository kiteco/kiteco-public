package account

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/hmacutil"
	"github.com/kiteco/kiteco/kite-golib/mixpanel"
)

const (
	timeout      = time.Minute * 3
	delightedURL = "https://api.delighted.com/v1/people.json"
)

type delightedManager struct {
	apiSecret string
	client    *http.Client
}

// DelightedPeople reflects the data structure required to send a survey using Delighted
type DelightedPeople struct {
	Email      string                 `json:"email"`
	Properties map[string]interface{} `json:"properties"`
}

// DelightedPerson is an internal data structure defined by Delighted's webhook
type DelightedPerson struct {
	Email string `json:"email"`
}

// DelightedEventData is an internal data structure defined by Delighted's webhook
type DelightedEventData struct {
	Person  DelightedPerson `json:"person"`
	Score   int             `json:"score"`
	Comment string          `json:"comment"`
}

// DelightedEvent is a data structure defined by Delighted's webhook
type DelightedEvent struct {
	EventData DelightedEventData `json:"event_data"`
	EventType string             `json:"event_type"`
	EventID   string             `json:"event_id"`
}

func newDelightedManager(secret string) *delightedManager {
	return &delightedManager{
		apiSecret: secret,
		client:    &http.Client{Timeout: timeout}, // generous timeout in case
	}
}

func (m *delightedManager) request(r *http.Request) (*http.Response, error) {
	r.SetBasicAuth(m.apiSecret, "")
	return m.client.Do(r)
}

func createPeopleRequest(people DelightedPeople) (*http.Request, error) {
	return http.NewRequest("POST", delightedURL, bytes.NewBuffer([]byte(people.encode())))
}

func createDelightedPeople(email, language string, mixpanel mixpanel.User) DelightedPeople {
	if mixpanel.DistinctID == "" {
		return DelightedPeople{
			Email: email,
		}
	}

	var os string
	if os, _ = mixpanel.Properties["os"].(string); os == "" {
		os, _ = mixpanel.Properties["$os"].(string)
	}

	people := DelightedPeople{
		Email: email,
		Properties: map[string]interface{}{
			"user_id":      mixpanel.DistinctID,
			"country":      mixpanel.Properties["$country_code"],
			"channel":      mixpanel.Properties["channel"],
			"kite_local":   mixpanel.Properties["kite_local"],
			"utm_source":   mixpanel.Properties["utm_source"],
			"utm_medium":   mixpanel.Properties["utm_medium"],
			"utm_campaign": mixpanel.Properties["utm_campaign"],
			"utm_term":     mixpanel.Properties["utm_term"],
			"utm_content":  mixpanel.Properties["utm_content"],
			"os":           os,
		},
	}

	if language != "" {
		people.Properties["language"] = language
	}

	return people
}

func (m *delightedManager) sendDelightedSurvey(people DelightedPeople) error {
	request, err := createPeopleRequest(people)
	if err != nil {
		return err
	}

	response, err := m.request(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return fmt.Errorf("unable to send survey to delighted: received status (%s)", response.Status)
	}

	return nil
}

func (d DelightedPeople) encode() (payload string) {
	payload = fmt.Sprintf("email=%s", url.QueryEscape(d.Email))

	for k, v := range d.Properties {
		k = url.QueryEscape(k)
		v = url.QueryEscape(fmt.Sprintf("%v", v))
		payload = fmt.Sprintf("%s&properties[%s]=%s", payload, k, v)
	}

	return
}

func (m *delightedManager) verifyWebhook(r *http.Request) (bool, error) {
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)
	r.Body = ioutil.NopCloser(&buf)

	message, err := ioutil.ReadAll(tee)
	if err != nil {
		return false, err
	}

	signature := r.Header["X-Delighted-Webhook-Signature"]
	if len(signature) != 1 {
		return false, fmt.Errorf("Delighted sent back more than one signature")
	}
	s := strings.Split(signature[0], "=")
	if len(s) != 2 {
		return false, fmt.Errorf("Delighted sent back the wrong signature format")
	}
	algorithm, messageMAC := s[0], s[1]
	if algorithm != "sha256" {
		return false, fmt.Errorf("Delighted sent back wrong hash algorithm")
	}

	digest, err := hex.DecodeString(messageMAC)
	if err != nil {
		return false, err
	}

	return hmacutil.CheckMAC(message, digest, []byte(m.apiSecret)), nil
}
