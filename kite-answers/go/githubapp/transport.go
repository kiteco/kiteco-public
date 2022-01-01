package githubapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

type jsonTime time.Time

func (j *jsonTime) UnmarshalJSON(data []byte) error {
	var timeStr string
	json.Unmarshal(data, &timeStr)
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return err
	}
	*j = jsonTime(t)
	return nil
}

type accessToken struct {
	Token     string   `json:"token"`
	ExpiresAt jsonTime `json:"expires_at"`
}

type installRoundTripper struct {
	creds     Credentials
	installID string

	base    http.RoundTripper
	baseCli http.Client

	lock  sync.Mutex
	token accessToken
}

func newInstallRoundTripper(creds Credentials, installID string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	baseCli := http.Client{Transport: base}

	return &installRoundTripper{
		creds:     creds,
		installID: installID,

		base:    base,
		baseCli: baseCli,
	}
}

func (a *installRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := a.getToken()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token.Token))
	return a.base.RoundTrip(req)
}

func (a *installRoundTripper) getToken() (accessToken, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.token.Token == "" || time.Until(time.Time(a.token.ExpiresAt)) < time.Minute {
		token, err := a.requestToken()
		if err != nil {
			return accessToken{}, nil
		}
		a.token = token
	}
	if a.token.Token == "" {
		return accessToken{}, errors.Errorf("could not fetch access token")
	}
	return a.token, nil
}

func (a *installRoundTripper) requestToken() (accessToken, error) {
	now := time.Now()
	claims := jwt.StandardClaims{
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(9 * time.Minute).Unix(),
		Issuer:    a.creds.ID,
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	jwtString, err := jwtToken.SignedString(a.creds.PrivateKey)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", a.installID), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtString))
	req.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
	resp, err := a.baseCli.Do(req)
	if err != nil {
		return accessToken{}, err
	}
	defer resp.Body.Close()

	var token accessToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return accessToken{}, err
	}
	return token, nil
}
