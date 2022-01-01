package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/version"
)

// blob returned from http://pypi.python.org/pypi/PACKAGE/json
type blob struct {
	Versions map[string][]struct{} `json:"releases"`
}

func getLatestVersionFor(d keytypes.Distribution) (string, error) {
	var latest version.Info
	if d.Version != "" {
		var err error
		latest, err = version.Parse(d.Version)
		if err != nil {
			log.Printf("failed to parse existing distribution version for %s", d.String())
			return d.Version, nil
		}
	}

	url := fmt.Sprintf("https://pypi.python.org/pypi/%s/json", d.Name)
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrapf(err, "error fetching packgage information for %s", d.Name)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", errors.Wrapf(err, "error reading body for package %s", d.Name)
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("got status code %d for package %s: %s", resp.StatusCode, d.Name, string(buf))
	}

	var b blob
	if err := json.Unmarshal(buf, &b); err != nil {
		return "", errors.Wrapf(err, "error decoding json response for %s", d.Name)
	}

	for v, rs := range b.Versions {
		if len(rs) == 0 {
			continue // no release candidates for this version...
		}
		parsed, err := version.Parse(v)
		if err != nil {
			log.Printf("failed to parse version identifier %s", v)
		}
		if parsed.LargerThanOrEqualTo(latest) {
			latest = parsed
		}
	}

	if v := latest.String(); v != "" {
		return v, nil
	}
	return "", errors.Errorf("no version found with release candidates")
}
