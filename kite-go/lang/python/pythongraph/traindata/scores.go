package traindata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// Errors maps from a symbol string to the error that
// occured while getting its score
type Errors map[string]string

// Err implements the error interface
func (es Errors) Err() string {
	var msgs []string
	for s, e := range es {
		msgs = append(msgs, fmt.Sprintf("%s: %s", s, e))
	}
	return strings.Join(msgs, ",")
}

type scoresRequest struct {
	Symbols      []string                 `json:"symbols"`
	Context      pythoncode.SymbolContext `json:"context"`
	Canonicalize bool                     `json:"canonicalize"`
}

type scoresResponse struct {
	Scores map[string]int    `json:"scores"`
	Errors map[string]string `json:"errors"`
}

// GetScores from the specified endpoint for the provided symbols,
// The function returns:
// - the scores
// - a map from symbol to error if there was an error getting a score for a particular symbol
// - an error if something catastrophic happened
func GetScores(endpoint string, symbols []pythonresource.Symbol, canonicalize bool, sc pythoncode.SymbolContext) (SymbolDist, Errors, error) {
	req := scoresRequest{
		Symbols:      make([]string, 0, len(symbols)),
		Context:      sc,
		Canonicalize: canonicalize,
	}

	for _, f := range symbols {
		req.Symbols = append(req.Symbols, f.PathString())
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return nil, nil, fmt.Errorf("error encoding request: %v", err)
	}

	resp, err := http.Post(endpoint, "application/json", &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := ioutil.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("got code %d: %v", resp.StatusCode, string(buf))
	}

	var sr scoresResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, nil, fmt.Errorf("error decoding response: %v", err)
	}

	dist := make(SymbolDist)
	for sym, count := range sr.Scores {
		dist[sym] = &SymbolDistEntry{
			Symbol:       sym,
			Canonicalize: canonicalize,
			Weight:       float64(count),
		}
	}

	return dist, Errors(sr.Errors), nil
}
