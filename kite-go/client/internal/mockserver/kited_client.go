package mockserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/community/account"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

var (
	// ErrUnauthorized is returned by CurrentUser() if the there's no logged in user available
	ErrUnauthorized = fmt.Errorf("user not logged in (unauthorized)")
)

// KitedClient is a client to talk to Kited
type KitedClient struct {
	URL        *url.URL
	httpClient *http.Client
}

// NewKitedClient returns a new kited client to talk to a server at kitedURL
func NewKitedClient(kitedURL *url.URL) *KitedClient {
	return &KitedClient{
		URL: kitedURL,
		httpClient: &http.Client{
			// don't follow redirects in this client
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// SendAccountCreationRequest posts the email and password to the server. It optionally waits until the login has been processed by the mockserver
func (t *KitedClient) SendAccountCreationRequest(email, password string, waitForCreation bool) (*http.Response, error) {
	return t.handleAccountCreationRequest("/clientapi/create-account", email, password, "", true, waitForCreation)
}

// SendPasswordlessAccountCreationRequest posts the email to the server. It optionally waits until the login has been processed by the mockserver
func (t *KitedClient) SendPasswordlessAccountCreationRequest(email, channel string, waitForCreation bool) (*http.Response, error) {
	return t.handleAccountCreationRequest("/clientapi/create-passwordless", email, "", channel, false, waitForCreation)
}

// handleAccountCreationRequest posts the email and an password to the server. The password is only send if sendPassword is true. It optionally waits until the login has been processed by the mockserver
func (t *KitedClient) handleAccountCreationRequest(urlPath, email, password, channel string, sendPassword, waitForCreation bool) (*http.Response, error) {
	requestURL, err := t.URL.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("email", email)
	w.WriteField("channel", channel)
	if sendPassword {
		w.WriteField("password", password)
	}
	w.Close()

	req, err := http.NewRequest("POST", requestURL.String(), body)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return resp, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK && waitForCreation {
		if err = t.waitForUser(email); err != nil {
			return nil, err
		}
	}

	return resp, err
}

// SendLoginRequest posts the email and password to the server. It optionally waits until the login has been processed by the mockserver
func (t *KitedClient) SendLoginRequest(email, password string, waitForLogin bool) (*http.Response, error) {
	requestURL, err := t.URL.Parse("/clientapi/login")
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("email", email)
	w.WriteField("password", password)
	w.Close()

	req, err := http.NewRequest("POST", requestURL.String(), body)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return resp, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK && waitForLogin {
		if err = t.waitForUser(email); err != nil {
			return resp, err
		}
	}

	return resp, err
}

// SendLogoutRequest sends a logout request to kited
func (t *KitedClient) SendLogoutRequest(waitForLogout bool) (*http.Response, error) {
	requestURL, err := t.URL.Parse("/clientapi/logout")
	if err != nil {
		return nil, err
	}

	resp, err := t.httpClient.Post(requestURL.String(), "application/json", nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if waitForLogout {
		if err = t.waitForUser(""); err != nil {
			return resp, err
		}
	}

	return resp, err
}

// CurrentUser retrieves information about the current user in kited
func (t *KitedClient) CurrentUser() (*community.User, error) {
	user := &community.User{}
	if err := t.GetJSON("/clientapi/user", user); err != nil {
		return nil, err
	}

	return user, nil
}

// CurrentPlan retrieves information about the current user's plan in kited
func (t *KitedClient) CurrentPlan() (*account.PlanResponse, error) {
	plan := &account.PlanResponse{}
	if err := t.GetJSON("/clientapi/plan", plan); err != nil {
		return nil, err
	}

	return plan, nil
}

// CurrentLicenseInfo retrieves information about the current user's plan in kited
func (t *KitedClient) CurrentLicenseInfo() (*licensing.LicenseInfo, error) {
	product := &licensing.LicenseInfo{}
	if err := t.GetJSON("/clientapi/license-info", product); err != nil {
		return nil, err
	}

	return product, nil
}

// GetJSON sends a HTTP GET with the given path to kited and parses the response data into target.
// It assumes that the response returns data as application/json
func (t *KitedClient) GetJSON(path string, target interface{}) error {
	resp, err := t.Get(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.Errorf("GET %s returned non-200 status: %d, message: %s, body: %s", path, resp.StatusCode, resp.Status, string(body))
	}

	return t.unmarshal(resp, target)
}

// Get sends a HTTP GET with path to kited
func (t *KitedClient) Get(path string) (*http.Response, error) {
	requestURL, err := t.URL.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return t.httpClient.Do(req)
}

// PostJSON sends a HTTP POST with path to kited and parsed the response body into target
// It assumes that the response returns data as application/json
func (t *KitedClient) PostJSON(path string, body io.Reader, target interface{}) (*http.Response, error) {
	resp, err := t.Post(path, body)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return resp, fmt.Errorf("POST %s returned non-200 status: %d", path, resp.StatusCode)
	}

	return resp, t.unmarshal(resp, target)
}

// Post sends a HTTP POST to kited
// path defines the URL's path, body provides the content of the request's body
func (t *KitedClient) Post(path string, body io.Reader) (*http.Response, error) {
	requestURL, err := t.URL.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", requestURL.String(), body)
	if err != nil {
		return nil, err
	}

	return t.httpClient.Do(req)
}

// PutJSON sends a HTTP PUT to kited and parses the response into target
// body defines the body of the request
func (t *KitedClient) PutJSON(path string, body io.Reader, target interface{}) error {
	resp, err := t.Put(path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("PUT %s returned non-200 status: %d", path, resp.StatusCode)
	}

	return t.unmarshal(resp, target)
}

// Put sends a HTTP PUT to kited
// body defines the request body
func (t *KitedClient) Put(path string, body io.Reader) (*http.Response, error) {
	requestURL, err := t.URL.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", requestURL.String(), body)
	if err != nil {
		return nil, err
	}

	return t.httpClient.Do(req)
}

// DeleteJSON sends a HTTP DELETE to kited and parses the response into target
// body defines the body of the request
func (t *KitedClient) DeleteJSON(path string, body io.Reader, target interface{}) error {
	resp, err := t.Delete(path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("PUT %s returned non-200 status: %d", path, resp.StatusCode)
	}

	return t.unmarshal(resp, target)
}

// Delete sends a HTTP DELETE to kited
func (t *KitedClient) Delete(path string, body io.Reader) (*http.Response, error) {
	requestURL, err := t.URL.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("DELETE", requestURL.String(), body)
	if err != nil {
		return nil, err
	}

	return t.httpClient.Do(req)
}

// Do executes a HTTP request with the configured HTTP client
func (t *KitedClient) Do(req *http.Request) (*http.Response, error) {
	return t.httpClient.Do(req)
}

// FileStatus returns the file's indexing status, it calls /clientapi/status?filename=$filename
func (t *KitedClient) FileStatus(file string) (string, error) {
	var response struct {
		Status string `json:"status"`
	}

	if err := t.GetJSON(fmt.Sprintf("/clientapi/status?filename=%s&checkloaded=false", file), &response); err != nil {
		return "", err
	}

	return response.Status, nil
}

// EditorBufferRequest defines all properties used by KitedClient.EditorBufferRequest().
type EditorBufferRequest struct {
	RequestType  string
	Editor       string
	File         string
	Hash         string
	FileContent  string
	CursorOffset int64
	UseRunes     bool
}

// EditorBufferRequest requests data from /api/.../$requestType, e.g. /api/.../hover
func (t *KitedClient) EditorBufferRequest(r EditorBufferRequest, target interface{}) (*http.Response, error) {
	unitPath, err := localpath.ToUnix(r.File)
	if err != nil {
		return nil, err
	}

	escapedPath := strings.Replace(unitPath, "/", ":", -1)

	var cursorParam string
	if r.UseRunes {
		cursorParam = "cursor_runes"
	} else {
		cursorParam = "cursor_bytes"
	}

	// attach current file content when necessary
	var body io.Reader
	if r.FileContent != "" {
		type requestBuffer struct {
			Buffer string `json:"buffer"`
		}

		b, err := json.Marshal(requestBuffer{Buffer: r.FileContent})
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}

	urlPath := path.Join("/api/buffer", r.Editor, escapedPath, r.Hash, r.RequestType)
	query := fmt.Sprintf("%s?%s=%d", urlPath, cursorParam, r.CursorOffset)
	return t.PostJSON(query, body, target)
}

// SymbolReport returns the symbol report for the given id
func (t *KitedClient) SymbolReport(id string) (*editorapi.ReportResponse, error) {
	response := editorapi.ReportResponse{}
	if err := t.GetJSON(fmt.Sprintf("/api/editor/symbol/%s", id), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Members returns a list of members for the given id
func (t *KitedClient) Members(id string) (*editorapi.MembersResponse, error) {
	response := editorapi.MembersResponse{}
	if err := t.GetJSON(fmt.Sprintf("/api/editor/value/%s/members", id), &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (t *KitedClient) unmarshal(response *http.Response, target interface{}) error {
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, target)
}

// waitForUser waits until a user with the given email address is logged in
// an empty email address will wait until no user is logged in
func (t *KitedClient) waitForUser(email string) error {
	for i := 0; i < 40; i++ { // wait up to 10s
		user, err := t.CurrentUser()
		if (email == "" && (err != nil || user == nil)) || (email != "" && err == nil && user.Email == email) {
			return nil
		}
		time.Sleep(time.Millisecond * 250)
	}

	user, _ := t.CurrentUser()
	if email == "" {
		return fmt.Errorf("error waiting for user logout, current user: %v", user)
	}
	return fmt.Errorf("error waiting for user login. expected: %s, current: %v", email, user)
}
