//go:generate go-bindata -pkg $GOPACKAGE -o bindata.go scripts/

package mixpanel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	timeout     = time.Minute * 3
	mixpanelURL = "https://mixpanel.com/api/2.0/jql"
)

// JQLClient is a client to query mixpanel for event and people information
type JQLClient struct {
	secret string
	client *http.Client
}

// NewJQLClient creates a new JQLClient
func NewJQLClient(secret string) *JQLClient {
	return &JQLClient{
		secret: secret,
		client: &http.Client{Timeout: timeout}, // generous timeout in case mixpanel is slow
	}
}

func (j *JQLClient) request(r *http.Request) (*http.Response, error) {
	r.SetBasicAuth(j.secret, "")
	return j.client.Do(r)
}

func createJQLRequest(script string) (*http.Request, error) {
	encodedScript := []byte(fmt.Sprintf("script=%s", url.QueryEscape(script)))
	return http.NewRequest("POST", mixpanelURL, bytes.NewBuffer(encodedScript))
}

func createUserJQL(email string) (string, error) {
	script, err := Asset("scripts/users.js")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(string(script), email), nil
}

// Query executes a JQL query and returns the raw JSON-encoded output.
func (j *JQLClient) Query(jql string) ([]byte, error) {
	request, err := createJQLRequest(jql)
	if err != nil {
		return nil, err
	}

	response, err := j.request(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read Mixpanel response body, status = %s: %v", response.Status, err)
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unable to execute Mixpanel JQL query, status = %s, body = %s",
			response.Status, string(body))
	}

	return body, nil
}

// RequestUserInfo queries mixpanel for an array of user properties under an email
func (j *JQLClient) RequestUserInfo(email string) ([]User, error) {
	jql, err := createUserJQL(email)
	if err != nil {
		return nil, err
	}

	resp, err := j.Query(jql)
	if err != nil {
		return nil, err
	}

	var users []User
	if err = json.Unmarshal([]byte(resp), &users); err != nil {
		return nil, err
	}

	return users, nil
}
