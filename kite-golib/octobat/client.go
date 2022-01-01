package octobat

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// NewBeanieClient returns a new client, which uses the defined secret key
func NewBeanieClient(octobatSecretKey string) *BeanieClient {
	return &BeanieClient{
		secretKey:  octobatSecretKey,
		httpClient: http.Client{},
	}
}

// BeanieClient interacts with the Octobat API
type BeanieClient struct {
	secretKey  string
	httpClient http.Client
}

// CreateSession POSTs to /beanie/sessions
func (c *BeanieClient) CreateSession(configuration BeanieServerSession) (BeanieSessionResponse, error) {
	body, err := json.Marshal(configuration)
	if err != nil {
		return BeanieSessionResponse{}, err
	}

	req, err := http.NewRequest("POST", "https://apiv2.octobat.com/beanie/sessions", bytes.NewReader(body))
	if err != nil {
		return BeanieSessionResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.secretKey, "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return BeanieSessionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BeanieSessionResponse{}, errors.New("Unexpected status code returned by server. Status code: %d, status text: %s", resp.StatusCode, resp.Status)
	}

	jsonResp := BeanieSessionResponse{}
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return BeanieSessionResponse{}, err
	}

	return jsonResp, nil
}
