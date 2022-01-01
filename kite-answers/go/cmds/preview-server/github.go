package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-answers/go/execution"
	"github.com/kiteco/kiteco/kite-answers/go/githubapp"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/shurcooL/githubv4"
)

const (
	ghAppID     = "XXXXXXX"
	ghInstallID = "XXXXXXX" // hard code this for convenience, rather than getting it from a webhook
	ghAppKey    = `-----BEGIN RSA PRIVATE KEY-----
XXXXXXX
-----END RSA PRIVATE KEY-----`
)

const (
	ghOrg     = "kite-answers"
	ghRepo    = "answers"
	ghOrgRepo = ghOrg + "/" + ghRepo
)

func handleGitHub(router *mux.Router, sandbox execution.Manager,
	resourceMgr pythonresource.Manager, app http.Handler) error {
	creds, err := githubapp.ParseCredentials(ghAppID, []byte(ghAppKey))
	if err != nil {
		return err
	}

	h := &githubHandler{
		sandbox:     sandbox,
		resourceMgr: resourceMgr,
		client:      githubapp.NewInstallClient(creds, ghInstallID, nil),
		app:         app,
	}
	router.Path("/github/").HandlerFunc(h.handleBookmarklet)
	router.Path("/github/pull/{pullNumber}/render").HandlerFunc(h.handlePull)
	router.Path("/github/pull/{pullNumber}/").Handler(h.app)
	router.Path("/github/blob/{commitish}/{path}/render").HandlerFunc(h.handleRender)
	router.Path("/github/blob/{commitish}/{path}/").Handler(h.app)

	return nil
}

type githubHandler struct {
	sandbox     execution.Manager
	resourceMgr pythonresource.Manager
	client      *githubv4.Client
	app         http.Handler
}

func (h *githubHandler) handleBookmarklet(w http.ResponseWriter, r *http.Request) {
	urls := r.URL.Query()["url"]
	if len(urls) == 0 {
		http.Error(w, "no url provided", http.StatusNotFound)
		return
	}
	uri, err := url.Parse(urls[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if uri.Host != "github.com" || err != nil {
		http.Error(w, "unrecognized url", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(uri.Path[1:], ghOrgRepo) || len(uri.Path) < 2+len(ghOrgRepo) || uri.Path[1+len(ghOrgRepo)] != '/' {
		http.Error(w, "unrecognized url", http.StatusBadRequest)
		return
	}
	path := uri.Path[2+len(ghOrgRepo):]

	switch {
	case strings.HasPrefix(path, "pull/"):
		path = strings.TrimPrefix(path, "pull/")
		parts := strings.Split(path, "/")
		pullNumber, err := strconv.Atoi(parts[0])
		if err != nil {
			http.Error(w, "failed to parse pull request number", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/github/pull/%d/", pullNumber), http.StatusFound)
		return
	case strings.HasPrefix(path, "blob/"):
		path = strings.TrimPrefix(path, "blob/")
		parts := strings.SplitN(path, "/", 2)
		commitish := parts[0]
		path := parts[1]
		http.Redirect(w, r, fmt.Sprintf("/github/blob/%s/%s/", commitish, url.QueryEscape(path)), http.StatusFound)
		return
	default:
		http.Error(w, "unrecognized url", http.StatusBadRequest)
		return
	}

}

func (h *githubHandler) handlePull(w http.ResponseWriter, r *http.Request) {
	muxVars := mux.Vars(r)
	pullNumber, err := strconv.Atoi(muxVars["pullNumber"])
	if err != nil {
		http.Error(w, "could not unescape path fragment", http.StatusBadRequest)
		return
	}

	var query struct {
		Repository struct {
			PullRequest struct {
				HeadRef struct {
					Target struct {
						OID githubv4.String `graphql:"oid"`
					}
				}
				Files struct {
					Nodes []struct {
						Path githubv4.String
					}
				} `graphql:"files(first: $numFiles)"`
			} `graphql:"pullRequest(number: $pullNumber)"`
		} `graphql:"repository(owner: $org, name: $repo)"`
	}
	vars := map[string]interface{}{
		"org":        githubv4.String(ghOrg),
		"repo":       githubv4.String(ghRepo),
		"pullNumber": githubv4.Int(pullNumber),
		"numFiles":   githubv4.Int(10),
	}

	if err := h.client.Query(context.TODO(), &query, vars); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	commitish := string(query.Repository.PullRequest.HeadRef.Target.OID)
	var b bytes.Buffer

	fmt.Fprintf(&b, "### Files Changed By [#%d](https://github.com/%s/pull/%d)\n\n", pullNumber, ghOrgRepo, pullNumber)
	for _, file := range query.Repository.PullRequest.Files.Nodes {
		path := string(file.Path)
		fmt.Fprintf(&b, " - [%s](/github/blob/%s/%s/)\n", path, commitish, url.QueryEscape(path))
	}

	doRender(w, h.sandbox, h.resourceMgr, b.Bytes())
}

func (h *githubHandler) handleRender(w http.ResponseWriter, r *http.Request) {
	muxVars := mux.Vars(r)
	path, err := url.QueryUnescape(muxVars["path"])
	if err != nil {
		http.Error(w, "could not unescape path fragment", http.StatusBadRequest)
		return
	}
	commitish := muxVars["commitish"]

	var query struct {
		Repository struct {
			Object struct {
				Blob struct {
					Text githubv4.String
				} `graphql:"... on Blob"`
			} `graphql:"object(expression: $objExpr)"`
		} `graphql:"repository(owner: $org, name: $repo)"`
	}
	vars := map[string]interface{}{
		"org":     githubv4.String(ghOrg),
		"repo":    githubv4.String(ghRepo),
		"objExpr": githubv4.String(fmt.Sprintf("%s:%s", commitish, path)),
	}

	if err := h.client.Query(context.TODO(), &query, vars); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mdText := []byte(string(query.Repository.Object.Blob.Text))
	doRender(w, h.sandbox, h.resourceMgr, mdText)
}
