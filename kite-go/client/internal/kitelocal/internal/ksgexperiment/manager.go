package ksgexperiment

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
)

var (
	autoSuggestURL  = "https://api.cognitive.microsoft.com/bing/v7.0/Suggestions"
	autoSuggestKey  = "XXXXXXX"
	googleSearchURL = "http://www.google.com/search"
	soQuestionLimit = "3"
	soURL           = "https://api.stackexchange.com/2.2"
	site            = "stackoverflow"
	// this will include the question title and the body in the stackexchange response (they're filtered by default)
	soFilter = "!-*jbN.OXKfDP"
)

// Manager registers endpoints for ksg completions and code blocks
type Manager struct {
	permissions component.PermissionsManager
	cohort      component.CohortManager
	client      *http.Client
}

// NewManager creates a new Manager
func NewManager() *Manager {
	return &Manager{
		client: &http.Client{},
	}
}

// Name implements component.Core
func (m *Manager) Name() string {
	return "ksg experiment"
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.permissions = opts.Permissions
	m.cohort = opts.Cohort
}

// RegisterHandlers implements component.Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/ksg/completions", m.permissions.WrapAuthorizedFile(m.cohort.WrapFeatureEnabled(m.handleCompletions)))
	mux.HandleFunc("/clientapi/ksg/codeblocks", m.permissions.WrapAuthorizedFile(m.cohort.WrapFeatureEnabled(m.handleCodeBlocks)))
}

// --

// AutosuggestResult contains the Bing Autosuggest response
type AutosuggestResult struct {
	SuggestionGroups []SuggestionGroup `json:"suggestionGroups"`
}

// SuggestionGroup contains the Bing Autosuggest suggestions
type SuggestionGroup struct {
	SearchSuggestions []SearchSuggestion `json:"searchSuggestions"`
}

// SearchSuggestion is a Bing Autosuggest suggestion
type SearchSuggestion struct {
	DisplayText string `json:"displayText"`
}

// CompletionsResponse contains the completions for the given query
type CompletionsResponse struct {
	Query       string   `json:"query"`
	Completions []string `json:"completions"`
}

func (m *Manager) handleCompletions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query was empty", http.StatusBadRequest)
		return
	}

	vals := url.Values{}
	vals.Set("mkt", "en-US")
	vals.Set("q", strings.TrimSpace(query))

	req, err := http.NewRequest("GET", autoSuggestURL+"?"+vals.Encode(), nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("error creating request for query %s: %v", query, err), http.StatusInternalServerError)
		return
	}
	req.Header.Add("Ocp-Apim-Subscription-Key", autoSuggestKey)

	resp, err := m.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting suggestions for query %s: %v", query, err), http.StatusInternalServerError)
		return
	}

	var result AutosuggestResult
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&result); err != nil {
		http.Error(w, fmt.Sprintf("error decoding response for query %s: %v", query, err), http.StatusInternalServerError)
		return
	}

	response := &CompletionsResponse{
		Query: query,
	}
	for _, sg := range result.SuggestionGroups {
		for _, ss := range sg.SearchSuggestions {
			response.Completions = append(response.Completions, ss.DisplayText)
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CodeBlocksResponse contains the code blocks for the given query
type CodeBlocksResponse struct {
	Query   string    `json:"query"`
	Answers []*Answer `json:"answers"`
}

func (m *Manager) handleCodeBlocks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query was empty", http.StatusBadRequest)
		return
	}
	desiredResults := r.URL.Query().Get("results")
	if desiredResults == "" {
		desiredResults = soQuestionLimit
	}

	questionCount, err := strconv.Atoi(desiredResults)
	if err != nil {
		http.Error(w, "results is not a number", http.StatusBadRequest)
		return
	}

	ids, err := m.searchGoogleForIDs(query, questionCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	answers, err := m.answersFromIDs(ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := &CodeBlocksResponse{
		Query:   query,
		Answers: answers,
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// --

// searchGoogleForIDs queries Google for StackOverflow results for the given query
// and returns question ids from the input count of search results
func (m *Manager) searchGoogleForIDs(query string, resultCount int) ([]string, error) {
	query = query + " site:stackoverflow.com"

	vals := url.Values{}
	vals.Set("q", strings.TrimSpace(query))
	vals.Set("num", strconv.Itoa(resultCount))
	url := googleSearchURL + "?" + vals.Encode()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// see SO link: /questions/56330930/beautifulsoup-select-method-not-selecting-results-as-expected
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux i686) AppleWebKit/537.17 (KHTML, like Gecko) Chrome/24.0.1312.27 Safari/537.17")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	var ids []string

	doc.Find("div.r").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Find("a").First().Attr("href")
		if url == "" {
			log.Println("for query:", query, "url is empty. DOM changed?")
		}
		url = sanitizeURL(url)
		// ensure result is from stackoverflow
		if !strings.Contains(url, "stackoverflow.com") {
			return
		}
		id, err := idFromURL(url)
		if err != nil {
			return
		}
		ids = append(ids, strconv.FormatInt(id, 10))
	})

	return ids, nil
}

func idFromURL(u string) (int64, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return 0, err
	}
	var id int64
	var discard string
	_, err = fmt.Sscanf(parsedURL.Path, "/questions/%d/%s", &id, &discard)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func sanitizeURL(u string) string {
	if strings.HasPrefix(u, "/url?") {
		u = strings.TrimPrefix(u, "/url?")
		v, _ := url.ParseQuery(u)
		return v.Get("q")
	}
	return u
}

// SOAnswerResponse contains the Stackoverflow answers for a given question
type SOAnswerResponse struct {
	Items []SOAnswer `json:"items"`
}

// SOAnswer contains the Stackoverflow answer body
type SOAnswer struct {
	AnswerID   int    `json:"answer_id"`
	QuestionID int    `json:"question_id"`
	IsAccepted bool   `json:"is_accepted"`
	Score      int    `json:"score"`
	Title      string `json:"title"`
	Body       string `json:"body"`
}

// Answer contains the code blocks extracted from the given Stackoverflow answer
type Answer struct {
	AnswerID      int      `json:"answer_id"`
	QuestionID    int      `json:"question_id"`
	IsAccepted    bool     `json:"is_accepted"`
	Votes         int      `json:"votes"`
	QuestionTitle string   `json:"question_title"`
	CodeBlocks    []string `json:"code_blocks"`
}

func (m *Manager) answersFromIDs(ids []string) ([]*Answer, error) {
	vals := url.Values{}
	vals.Set("site", site)
	vals.Set("filter", soFilter)
	questionURL := soURL + "/questions/" + strings.Join(ids, ";") +
		"/answers" + "?" + vals.Encode()
	qURL, err := url.Parse(questionURL)
	if err != nil {
		return nil, fmt.Errorf("error getting answers: error parsing url")
	}
	req, err := http.NewRequest("GET", qURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error getting answers: error creating request")
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting answers: error making request")
	}

	var result SOAnswerResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&result); err != nil {
		return nil, fmt.Errorf("error getting answers: error decoding response: %v", err)
	}

	var answers []*Answer
	for _, item := range result.Items {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(item.Body))
		if err != nil {
			return nil, fmt.Errorf("error creating document: %v", err)
		}
		var codeBlocks []string
		// <pre><code></code></pre> structures are code blocks
		doc.Find("pre code").Each(func(i int, s *goquery.Selection) {
			codeBlocks = append(codeBlocks, s.Text())
		})
		answers = append(answers, &Answer{
			AnswerID:      item.AnswerID,
			QuestionID:    item.QuestionID,
			Votes:         item.Score,
			QuestionTitle: html.UnescapeString(item.Title), //unescape this
			IsAccepted:    item.IsAccepted,
			CodeBlocks:    codeBlocks,
		})
	}
	return answers, nil
}
