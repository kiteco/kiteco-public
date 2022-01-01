package stackoverflow

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
)

var (
	maxStackoverflowPostsPerError int
	maxStackoverflowPostsToFetch  int
)

// SearchType specifies the type of search performed by the index.
type SearchType string

const (
	// Disjunction performs a disjunctive query.
	Disjunction SearchType = "disj"
	// Conjunction performs a conjunctive query.
	Conjunction SearchType = "conj"
)

// SearchMode specifies which engine we should use for performing
// the search.
type SearchMode string

const (
	// Bing uses the Bing search API to return relevant pages.
	Bing SearchMode = "bing"
	// Google uses the Google search API to return relevant pages.
	Google SearchMode = "google"
	// Kite uses the Kite search API to return relevant pages.
	Kite SearchMode = "kite"
)

// ClientOptions specifies the options which parametrize a Client.
type ClientOptions struct {
	Mode       SearchMode
	SearchType SearchType
	Endpoint   string
}

var (
	// DefaultClientOptions specifies the default options for a Client.
	DefaultClientOptions = ClientOptions{
		Mode:       Kite,
		SearchType: Conjunction,
		Endpoint:   "http://search-0.kite.com:8090",
	}
)

// Client makes search requests to a search server.
type Client struct {
	serverURL  *url.URL
	httpClient *http.Client
	index      errorIndex
	mode       string
	searchType string
}

func init() {
	flag.IntVar(&maxStackoverflowPostsPerError, "maxStackoverflowPostsPerError", 2,
		"Limit on the number of stackoverflow posts to return per terminal error message")
	flag.IntVar(&maxStackoverflowPostsToFetch, "maxStackoverflowPostsToFetch", 25,
		"Limit on the number of stackoverflow posts to fetch from backend")
}

// NewClient returns a pointer to a newly initialized Client.
func NewClient(opts *ClientOptions) (*Client, error) {
	if opts == nil {
		opts = &ClientOptions{}
		*opts = DefaultClientOptions
	}
	endpoint := opts.Endpoint
	if endpoint == "" {
		endpoint = DefaultClientOptions.Endpoint
	}
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error parsing url: %s", err)
	}

	// Load the error index
	index, err := newDefaultErrorIndex()
	if err != nil {
		// Report the error but still go ahead and return a valid client
		log.Println("Failed to load error index: ", err)
		return nil, fmt.Errorf("failed to load error index: %s", err)
	}

	return &Client{
		serverURL:  url,
		httpClient: &http.Client{},
		index:      index,
		mode:       string(opts.Mode),
		searchType: string(opts.SearchType),
	}, nil
}

// Search searches for pages on StackOverflow relevant to the given query and returns the results.
func (c *Client) Search(query string, maxResults int) ([]*StackOverflowPage, error) {
	v := url.Values{}
	v.Set("q", query)
	v.Set("mr", strconv.Itoa(maxResults))
	v.Set("st", c.searchType)
	v.Set("mode", c.mode)

	soEndpoint, err := c.serverURL.Parse("/search" + "?" + v.Encode())
	if err != nil {
		return nil, fmt.Errorf("error parsing search url: %s", err)
	}

	return c.getJSON(soEndpoint)
}

// PostsByID returns StackOverflowPage's corersponding to the provided ids
func (c *Client) PostsByID(ids []int) ([]*StackOverflowPage, error) {
	var idstrs []string
	for _, id := range ids {
		idstrs = append(idstrs, strconv.Itoa(id))
	}

	v := url.Values{}
	v.Set("ids", strings.Join(idstrs, ","))
	soEndpoint, err := c.serverURL.Parse("/posts?" + v.Encode())
	if err != nil {
		return nil, fmt.Errorf("error parsing posts url: %s", err)
	}
	return c.getJSON(soEndpoint)
}

// PostsForError return StackOverflowPages corresponding to the provided language and error id.
func (c *Client) PostsForError(language lang.Language, errorID int) []*StackOverflowPage {
	// Lookup stackoverflow pages for this error
	postIds := c.index.LookupPosts(language, errorID)
	if len(postIds) == 0 {
		return nil
	}

	// Limit the number of posts to fetch from the search server
	if len(postIds) > maxStackoverflowPostsToFetch {
		// TODO(alex): sort the index by stackoverflow votes so that this
		// selects the top N pages.
		postIds = postIds[:maxStackoverflowPostsToFetch]
	}

	// Fetch full post info from search backend
	posts, err := c.PostsByID(postIds)
	if err != nil {
		// TODO(juan): return this error to caller
		log.Println(err)
	}

	// Sort posts by votes
	RankInPlace(posts)

	// Apply cutoff
	if len(posts) > maxStackoverflowPostsPerError {
		posts = posts[:maxStackoverflowPostsPerError]
	}

	return posts
}

func (c *Client) getJSON(endpoint *url.URL) ([]*StackOverflowPage, error) {
	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error constructing request: %s", err)
	}

	result, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to %v: %s", endpoint, err)
	}
	defer result.Body.Close()

	respBuf, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("error during HTTP GET: %s", err)
	}

	var results []*StackOverflowPage
	err = json.Unmarshal(respBuf, &results)
	return results, err
}
