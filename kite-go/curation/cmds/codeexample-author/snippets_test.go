package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserName     = "Kite"
	testUserEmail    = "test@kite.com"
	testUserPassword = "test123"

	testLang    = "python"
	testPackage = "os"
)

func TestCuratedSnippet_HandleCreateGet(t *testing.T) {
	ts, client, _ := buildServerClient()

	// Get access lock
	packageURL := makeTestURL(ts.URL, fmt.Sprintf("/api/%s/%s/lockAndList", testLang, testPackage))
	accessResp, err := client.Get(packageURL)
	assert.NoError(t, err, "expected package access request to succeed", packageURL)
	assert.Equal(t, http.StatusOK, accessResp.StatusCode)

	// Make a fake snippet, create it via API
	snippet1 := makeFakeSnippet(0)
	buf := marshal(snippet1)
	createURL := makeTestURL(ts.URL, fmt.Sprintf("/api/%s/%s/examples", testLang, testPackage))
	resp, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
	assert.NoError(t, err, "expected POST to %s to succeed", createURL)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// Retrieve the response, make sure snippet ID is set and that contents are equal to request
	snippet2 := &curation.CuratedSnippet{}
	err = unmarshal(resp.Body, snippet2)
	require.NoError(t, err)
	assert.True(t, snippet2.SnippetID > 0, "expected SnippetID to be set")
	assertSnippetsEqual(t, snippet1, snippet2)

	// Do a GET on the same snippetID to ensure we get the same snippet back
	getURL := makeTestURL(ts.URL, fmt.Sprintf("/api/example/%d", snippet2.SnippetID))
	resp2, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	defer resp2.Body.Close()

	// Make sure returned snippet is the same
	snippet3 := &curation.CuratedSnippet{}
	err = unmarshal(resp2.Body, snippet3)
	require.NoError(t, err)
	assert.Equal(t, snippet2.SnippetID, snippet3.SnippetID, "expected snippetID's to be equal")
	assertSnippetsEqual(t, snippet2, snippet3)
}

func TestCuratedSnippet_HandleGet_NotExist(t *testing.T) {
	ts, client, _ := buildServerClient()

	// Get access lock
	packageURL := makeTestURL(ts.URL, fmt.Sprintf("/api/%s/%s/lockAndList", testLang, testPackage))
	accessResp, err := client.Get(packageURL)
	assert.NoError(t, err, "expected package access request to succeed", packageURL)
	assert.Equal(t, http.StatusOK, accessResp.StatusCode)

	// Do a GET on a random snippetID that does not exist
	getURL := makeTestURL(ts.URL, fmt.Sprintf("/api/example/%d", 347))
	resp, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
}

func TestCuratedSnipet_HandleCreate_AlreadyExist(t *testing.T) {
	ts, client, _ := buildServerClient()

	// Get access lock
	packageURL := makeTestURL(ts.URL, fmt.Sprintf("/api/%s/%s/lockAndList", testLang, testPackage))
	accessResp, err := client.Get(packageURL)
	assert.NoError(t, err, "expected package access request to succeed", packageURL)
	assert.Equal(t, http.StatusOK, accessResp.StatusCode)

	// Make a fake snippet, create it via API
	snippet1 := makeFakeSnippet(0)
	createURL := makeTestURL(ts.URL, fmt.Sprintf("/api/%s/%s/examples", testLang, testPackage))
	buf := marshal(snippet1)
	resp, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
	assert.NoError(t, err, "expected post to %s to succeed", createURL)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	snippet2 := &curation.CuratedSnippet{}
	err = unmarshal(resp.Body, snippet2)
	require.NoError(t, err)

	// Do it again...
	buf = marshal(snippet2)
	resp2, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
	assert.NoError(t, err, "expected POST to %s to succeed", createURL)
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
	defer resp2.Body.Close()
	io.Copy(ioutil.Discard, resp2.Body)
}

func TestCuratedSnippet_HandleUpdate(t *testing.T) {
	ts, client, app := buildServerClient()

	snippet1 := makeFakeSnippet(0)
	err := app.Snippets.Create(snippet1)
	require.NoError(t, err, "expected fixture snippet creation to succeed")

	// Get the created snippet
	snippet2, err := app.Snippets.GetByID(snippet1.SnippetID)
	require.NoError(t, err, "expected fixture snippet retrieval to succeed")

	// Update the snippet copy
	snippet2.Title = snippet2.Title + "UPDATED"
	snippet2.Code = snippet2.Code + "UPDATED"
	snippet2.Prelude = snippet2.Prelude + "UPDATED"
	snippet2.Postlude = snippet2.Postlude + "UPDATED"

	// Get access lock
	packageURL := makeTestURL(ts.URL, fmt.Sprintf("/api/%s/%s/lockAndList", testLang, testPackage))
	accessResp, err := client.Get(packageURL)
	assert.NoError(t, err, "expected package access request to succeed", packageURL)
	assert.Equal(t, http.StatusOK, accessResp.StatusCode)

	// Make PUT request to update snippet on the backend
	updateURL := makeTestURL(ts.URL, fmt.Sprintf("/api/example/%d", snippet2.SnippetID))
	buf := marshal(snippet2)
	req, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(buf))
	assert.NoError(t, err, "error building request")

	resp2, err := client.Do(req)
	assert.NoError(t, err, "expected PUT to %s to succeed", updateURL)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Retrieve the snippet to make sure changes persist
	getURL := makeTestURL(ts.URL, fmt.Sprintf("/api/example/%d", snippet2.SnippetID))
	resp3, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	defer resp3.Body.Close()

	snippet3 := &curation.CuratedSnippet{}
	err = unmarshal(resp3.Body, snippet3)
	require.NoError(t, err)
	assertSnippetsEqual(t, snippet2, snippet3)
}

func TestCuratedSnippet_HandleUpdate_RequiresAccessLock(t *testing.T) {
	ts, client, app := buildServerClient()

	// Make a fake snippet
	snippet1 := makeFakeSnippet(0)
	err := app.Snippets.Create(snippet1)
	require.NoError(t, err, "expected fixture snippet creation to succeed")

	// Get the created snippet
	snippet2, err := app.Snippets.GetByID(snippet1.SnippetID)
	require.NoError(t, err, "expected fixture snippet retrieval to succeed")

	// Update the snippet
	snippet2.Title = snippet2.Title + "UPDATED"
	snippet2.Code = snippet2.Code + "UPDATED"
	snippet2.Prelude = snippet2.Prelude + "UPDATED"
	snippet2.Postlude = snippet2.Postlude + "UPDATED"

	// Uh oh -- someone else just acquired the access lock!
	app.Access.acquireAccessLock(testLang, testPackage, "not_"+testUserEmail)

	// Make PUT request to update snippet on the backend
	updateURL := makeTestURL(ts.URL, fmt.Sprintf("/api/example/%d", snippet2.SnippetID))
	buf := marshal(snippet2)
	req, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(buf))
	assert.NoError(t, err, "error building request")

	resp2, err := client.Do(req)
	assert.NoError(t, err, "expected PUT to %s to succeed", updateURL)
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

	// Retrieve the snippet to make sure changes persist
	getURL := makeTestURL(ts.URL, fmt.Sprintf("/api/example/%d", snippet2.SnippetID))
	resp3, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	defer resp3.Body.Close()

	snippet3 := &curation.CuratedSnippet{}
	err = unmarshal(resp3.Body, snippet3)
	require.NoError(t, err)
	assert.NotEqual(t, snippet2.Title, snippet3.Title)
	assert.NotEqual(t, snippet2.Code, snippet3.Code)
	assert.NotEqual(t, snippet2.Prelude, snippet3.Prelude)
	assert.NotEqual(t, snippet2.Postlude, snippet3.Postlude)
}

func TestCuratedSnippet_HandleList(t *testing.T) {
	ts, client, app := buildServerClient()

	// Make a bunch of snippets
	snippetsToCreate := 10
	var original []*curation.CuratedSnippet
	for i := 0; i < snippetsToCreate; i++ {
		snippet := makeFakeSnippet(i)
		original = append(original, snippet)
		err := app.Snippets.Create(snippet)
		assert.NoError(t, err, "expected fixture snippet creation to succeed")
	}

	listURL := makeTestURL(ts.URL, fmt.Sprintf("/api/%s/%s/examples", testLang, testPackage))
	resp, err := client.Get(listURL)
	assert.NoError(t, err, "expected GET to %s to succeed", listURL)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err, "error reading body of examples list response")

	var snippets []*curation.CuratedSnippet
	err = json.Unmarshal(buf, &snippets)
	assert.NoError(t, err, "error unmarshalling snippet list")

	assert.Equal(t, len(original), len(snippets), "got different number of snippets")
	for i := 0; i < len(snippets); i++ {
		assertSnippetsEqual(t, snippets[i], original[i])
	}
}

// --

func TestCuratedSnippet_HandleCreateGetComment(t *testing.T) {
	ts, client, _ := buildServerClient()
	comment1 := makeFakeComment(0)
	createURL := makeTestURL(ts.URL, "/api/example/1/comments")

	// Make a fake comment, create it using `POST` endpoint
	buf := marshal(comment1)
	resp, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
	assert.NoError(t, err, "expected POST to %s to succeed", createURL)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// Retrieve the response, make sure snippetID is set to 1 and that contents
	// are equal to request
	comment2 := &curation.Comment{}
	err = unmarshal(resp.Body, comment2)
	require.NoError(t, err)
	assert.True(t, comment2.ID > 0, "expected comment ID to be set")
	assert.EqualValues(t, 1, comment2.SnippetID, "expected SnippetID to be 1")
	assert.NotEqual(t, 0, comment2.Created, "expected Created field to be set")
	assert.NotEqual(t, "", comment2.CreatedBy, "expected Created field to be set")
	assert.Equal(t, comment1.Text, comment2.Text, "expected Text fields to be the same")

	// Do a GET on the same comment ID to ensure we get the same comment back
	getURL := makeTestURL(ts.URL, fmt.Sprintf("/api/comment/%d", comment2.ID))
	resp2, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	defer resp2.Body.Close()

	// Make sure returned comment is the same
	comment3 := &curation.Comment{}
	err = unmarshal(resp2.Body, comment3)
	require.NoError(t, err)
	assert.Equal(t, comment2.ID, comment3.ID, "expected comment ID's to be equal")
	assertCommentEqual(t, comment2, comment3)
}

func TestCuratedSnippet_HandleCreateComment_AlreadyExist(t *testing.T) {
	ts, client, _ := buildServerClient()
	comment1 := makeFakeComment(0)
	createURL := makeTestURL(ts.URL, "/api/example/1/comments")

	// Make a fake comment, create it using `POST` endpoint
	buf := marshal(comment1)
	resp, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
	assert.NoError(t, err, "expected POST to %s to succeed", createURL)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// Retrieve the response, make sure snippetID is set to 1 and that contents
	// are equal to request
	comment2 := &curation.Comment{}
	err = unmarshal(resp.Body, comment2)
	require.NoError(t, err)
	assert.True(t, comment2.ID > 0, "expected comment ID to be set")
	assert.EqualValues(t, 1, comment2.SnippetID, "expected SnippetID to be 1")
	assert.NotEqual(t, 0, comment2.Created, "expected Created field to be set")
	assert.NotEqual(t, "", comment2.CreatedBy, "expected Created field to be set")
	assert.Equal(t, comment1.Text, comment2.Text, "expected Text fields to be the same")

	// Do it again...
	buf = marshal(comment2)
	resp2, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
	assert.NoError(t, err, "expected POST to %s to succeed", createURL)
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
	defer resp2.Body.Close()
	io.Copy(ioutil.Discard, resp2.Body)
}

func TestCuratedSnippet_HandleGetComment_NotExist(t *testing.T) {
	ts, client, _ := buildServerClient()

	getURL := makeTestURL(ts.URL, "/api/comment/456")
	resp, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
}

func TestCuratedSnippet_HandleUpdateComment(t *testing.T) {
	ts, client, _ := buildServerClient()
	comment1 := makeFakeComment(0)
	createURL := makeTestURL(ts.URL, "/api/example/1/comments")

	// Make a fake comment, create it using `POST` endpoint
	buf := marshal(comment1)
	resp, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
	assert.NoError(t, err, "expected POST to %s to succeed", createURL)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// Retrieve the response, make sure snippetID is set to 1 and that contents
	// are equal to request
	comment2 := &curation.Comment{}
	err = unmarshal(resp.Body, comment2)
	require.NoError(t, err)
	assert.True(t, comment2.ID > 0, "expected comment ID to be set")
	assert.EqualValues(t, 1, comment2.SnippetID, "expected SnippetID to be 1")
	assert.NotEqual(t, 0, comment2.Created, "expected Created field to be set")
	assert.NotEqual(t, "", comment2.CreatedBy, "expected Created field to be set")
	assert.Equal(t, comment1.Text, comment2.Text, "expected Text fields to be the same")

	comment2.Text = comment2.Text + "UPDATED"

	updateURL := makeTestURL(ts.URL, fmt.Sprintf("/api/comment/%d", comment2.ID))
	buf = marshal(comment2)
	req, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(buf))
	assert.NoError(t, err, "error building PUT request")

	resp2, err := client.Do(req)
	assert.NoError(t, err, "expected PUT to %s to succeed", updateURL)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Do a GET on the same comment ID to ensure we get the updated comment back
	getURL := makeTestURL(ts.URL, fmt.Sprintf("/api/comment/%d", comment2.ID))
	resp3, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	defer resp3.Body.Close()

	// Make sure returned comment is the same
	comment3 := &curation.Comment{}
	err = unmarshal(resp3.Body, comment3)
	require.NoError(t, err)
	assert.Equal(t, comment2.ID, comment3.ID, "expected comment ID's to be equal")
	assertCommentEqual(t, comment2, comment3)
}

func TestCuratedSnippet_HandleUpdateComment_Dismiss(t *testing.T) {
	ts, client, _ := buildServerClient()
	comment1 := makeFakeComment(0)
	createURL := makeTestURL(ts.URL, "/api/example/1/comments")

	// Make a fake comment, create it using `POST` endpoint
	buf := marshal(comment1)
	resp, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
	assert.NoError(t, err, "expected POST to %s to succeed", createURL)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// Retrieve the response, make sure snippetID is set to 1 and that contents
	// are equal to request
	comment2 := &curation.Comment{}
	err = unmarshal(resp.Body, comment2)
	require.NoError(t, err)
	assert.True(t, comment2.ID > 0, "expected comment ID to be set")
	assert.EqualValues(t, 1, comment2.SnippetID, "expected SnippetID to be 1")
	assert.NotEqual(t, 0, comment2.Created, "expected Created field to be set")
	assert.NotEqual(t, "", comment2.CreatedBy, "expected Created field to be set")
	assert.Equal(t, comment1.Text, comment2.Text, "expected Text fields to be the same")

	comment2.Dismissed = 1 // dismiss the comment

	updateURL := makeTestURL(ts.URL, fmt.Sprintf("/api/comment/%d", comment2.ID))
	buf = marshal(comment2)
	req, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(buf))
	assert.NoError(t, err, "error building PUT request")

	resp2, err := client.Do(req)
	assert.NoError(t, err, "expected PUT to %s to succeed", updateURL)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Do a GET on the same comment ID to ensure we get the updated comment back
	getURL := makeTestURL(ts.URL, fmt.Sprintf("/api/comment/%d", comment2.ID))
	resp3, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
	defer resp3.Body.Close()

	// Make sure returned comment is the same
	comment3 := &curation.Comment{}
	err = unmarshal(resp3.Body, comment3)
	require.NoError(t, err)
	assert.Equal(t, comment2.ID, comment3.ID, "expected comment ID's to be equal")
	assert.True(t, comment3.Dismissed > 1, "expected Dismissed field to be set with a timestamp")

	// Try to modify dismissed comment
	comment3.Text = "NEW TEXT"

	buf = marshal(comment3)
	req, err = http.NewRequest("PUT", updateURL, bytes.NewBuffer(buf))
	assert.NoError(t, err, "error building PUT request")

	resp4, err := client.Do(req)
	assert.NoError(t, err, "expected PUT on dismissed comment to %s to succeed", updateURL)
	assert.Equal(t, http.StatusBadRequest, resp4.StatusCode)

	// Undismiss the comment
	comment3.Dismissed = 0

	buf = marshal(comment3)
	req, err = http.NewRequest("PUT", updateURL, bytes.NewBuffer(buf))
	assert.NoError(t, err, "error building PUT request")

	resp5, err := client.Do(req)
	assert.NoError(t, err, "expected PUT to undismiss comment to %s to succeed", updateURL)
	assert.Equal(t, http.StatusOK, resp5.StatusCode)

	// Do a GET on the same comment ID to ensure we get the updated comment back
	getURL = makeTestURL(ts.URL, fmt.Sprintf("/api/comment/%d", comment3.ID))
	resp6, err := client.Get(getURL)
	assert.NoError(t, err, "expected GET to %s to succeed", getURL)
	assert.Equal(t, http.StatusOK, resp6.StatusCode)
	defer resp6.Body.Close()

	// Make sure returned comment is the same
	comment4 := &curation.Comment{}
	err = unmarshal(resp6.Body, comment4)
	require.NoError(t, err)
	assert.Equal(t, comment3.ID, comment4.ID, "expected comment ID's to be equal")
	assert.True(t, comment4.Dismissed == 0, "expected Dismissed field to be set to zero")
}

func TestCuratedSnippet_HandleListComments(t *testing.T) {
	ts, client, _ := buildServerClient()
	commentsToCreate := 10

	// Make many comments
	var original []*curation.Comment
	createURL := makeTestURL(ts.URL, "/api/example/1/comments")
	for i := 0; i < commentsToCreate; i++ {
		comment := makeFakeComment(i)
		original = append(original, comment)

		// Make a fake comment, create it using `POST` endpoint
		buf := marshal(comment)
		resp, err := client.Post(createURL, "application/json", bytes.NewBuffer(buf))
		assert.NoError(t, err, "expected POST to %s to succeed", createURL)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()
		io.Copy(ioutil.Discard, resp.Body)
	}

	listURL := makeTestURL(ts.URL, "/api/example/1/comments")
	resp, err := client.Get(listURL)
	assert.NoError(t, err, "expected GET to %s to succeed", listURL)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err, "error ready body")

	var comments []*curation.Comment
	err = json.Unmarshal(buf, &comments)
	assert.NoError(t, err, "error unmarshalling comment list")

	assert.Equal(t, len(original), len(comments), "got different number of comments")
	for i := 0; i < len(comments); i++ {
		assert.Equal(t, comments[i].Text, original[i].Text, "expected Text field to be the same")
	}
}

// --

type mockInviteCodeEmailer struct {
	Addrs []string
}

func (m *mockInviteCodeEmailer) Render(host, inviteCode string) (*bytes.Buffer, error) {
	var body bytes.Buffer
	return &body, nil
}

func (m *mockInviteCodeEmailer) Email(addr string, body *bytes.Buffer) error {
	m.Addrs = append(m.Addrs, addr)
	return nil
}

func buildServerClient() (*httptest.Server, *http.Client, *App) {
	ts, app, authDB := makeTestServer()
	client := makeTestClient()
	authDB.DropTableIfExists(&community.User{})
	authDB.CreateTable(&community.User{})

	_, _, err := app.Users.Create(testUserName, testUserEmail, testUserPassword, "")
	if err != nil {
		log.Fatal(err)
	}

	creds := url.Values{}
	creds.Set("email", testUserEmail)
	creds.Set("password", testUserPassword)

	loginURL := makeTestURL(ts.URL, "/login")
	resp, err := client.PostForm(loginURL, creds)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	return ts, client, app
}

// setupSignupManagerWithDB takes a db and creates a new signup manager using it.
func setupSignupManagerWithDB(db gorm.DB) *community.SignupManager {
	db.DropTableIfExists(&community.Signup{})
	db.DropTableIfExists(&community.Download{})
	manager := community.NewSignupManager(db)
	if err := manager.Migrate(); err != nil {
		log.Fatalln(err)
	}
	return manager
}

func unmarshal(r io.Reader, obj interface{}) error {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &obj)
	if err != nil {
		return err
	}
	return nil
}

func marshal(obj interface{}) []byte {
	buf, err := json.Marshal(obj)
	if err != nil {
		log.Fatal(err)
	}
	return buf
}

func makeFakeComment(n int) *curation.Comment {
	return &curation.Comment{
		Text: fmt.Sprintf("comment%d", n),
	}
}

func makeFakeSnippet(n int) *curation.CuratedSnippet {
	return &curation.CuratedSnippet{
		Status:   curation.SnippetStatusInProgress,
		Language: testLang,
		Package:  testPackage,
		Title:    fmt.Sprintf("title%d", n),
		Code:     fmt.Sprintf("code%d", n),
		Prelude:  fmt.Sprintf("prelude%d", n),
		Postlude: fmt.Sprintf("postlude%d", n),
	}
}

func assertSnippetsEqual(t *testing.T, s1, s2 *curation.CuratedSnippet) {
	assert.Equal(t, s1.Status, s2.Status, "Snippets different on Status field")
	assert.Equal(t, s1.Language, s2.Language, "Snippets different on Language field")
	assert.Equal(t, s1.Package, s2.Package, "Snippets different on Package field")
	assert.Equal(t, s1.Title, s2.Title, "Snippets different on Title field")
	assert.Equal(t, s1.Code, s2.Code, "Snippets different on Code field")
	assert.Equal(t, s1.Prelude, s2.Prelude, "Snippets different on Prelude field")
	assert.Equal(t, s1.Postlude, s2.Postlude, "Snippets different on Postlude field")
	assertCommentsEqual(t, s1.Comments, s2.Comments)
	assertSupportingFilesEqual(t, s1.SupportingFiles, s2.SupportingFiles)
}

func assertCommentEqual(t *testing.T, comment1, comment2 *curation.Comment) {
	assert.Equal(t, comment1.Text, comment2.Text, "Comments different on Text")

	assert.Equal(t, comment1.Created, comment2.Created, "Comments different on Created")
	assert.Equal(t, comment1.CreatedBy, comment2.CreatedBy, "Comments different on CreatedBy")

	assert.Equal(t, comment1.Dismissed, comment2.Dismissed, "Comments different on Dismissed")
}

func assertCommentsEqual(t *testing.T, c1, c2 []*curation.Comment) {
	if !assert.Equal(t, len(c1), len(c2), "Snippets different on number of comments") {
		return
	}

	for i := 0; i < len(c1); i++ {
		assertCommentEqual(t, c1[i], c2[i])
	}
}

func assertSupportingFilesEqual(t *testing.T, f1, f2 []*curation.SupportingFile) {
	if !assert.Equal(t, len(f1), len(f2), "Snippets different on number of supporting files") {
		return
	}

	for i := 0; i < len(f1); i++ {
		assert.Equal(t, f1[i].Path, f2[i].Path, "Snippets different on Path for supporting file at index %d", i)
		assert.True(t, bytes.Equal(f1[i].Contents, f2[i].Contents), "Snippets different on Contents for supporting file at index %d", i)
	}
}

func pprint(data interface{}) {
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(buf))
}
