package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type client struct {
	endpoint string
}

func newClient(endpoint string) client {
	return client{
		endpoint: endpoint,
	}
}

func (c client) Start(req StartRequest) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}

	resp, err := http.Post(c.url("/api/start"), "application/json", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var body bytes.Buffer
		_, err = body.ReadFrom(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("error calling /api/start, status code %d: %s", resp.StatusCode, body.String())
	}

	return nil
}

func (c client) Status() (StatusResponse, error) {
	var status StatusResponse

	if err := c.getRequest("/api/status", &status); err != nil {
		return StatusResponse{}, err
	}

	return status, nil
}

func (c client) Results() (ResultsResponse, error) {
	var results ResultsResponse

	if err := c.getRequest("/api/results", &results); err != nil {
		return ResultsResponse{}, err
	}

	return results, nil
}

func (c client) FeedStats() (map[string]FeedStats, error) {
	var stats map[string]FeedStats
	if err := c.getRequest("/api/feed-stats", &stats); err != nil {
		return nil, err
	}
	return stats, nil
}

func (c client) url(relativePath string) string {
	return fmt.Sprintf("http://%s%s", c.endpoint, relativePath)
}

func (c client) getRequest(relativeURL string, toLoad interface{}) error {
	resp, err := http.Get(c.url(relativeURL))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body bytes.Buffer
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("error calling %s, status code %d: %s", relativeURL, resp.StatusCode, body.String())
	}

	if err := json.NewDecoder(&body).Decode(toLoad); err != nil {
		return err
	}

	return nil
}
