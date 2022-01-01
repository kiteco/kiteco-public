package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/lsp/types"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// Onboarding returns the onboarding file path for JupyterLab
func (h *Handlers) Onboarding(params types.KiteOnboardingParams) (string, error) {
	queryParams := map[string]string{
		"editor": data.JupyterEditor.String(),
	}
	onboardingURL, err := buildURL(onboardingURL, queryParams)
	if err != nil {
		return "", err
	}

	res, err := http.Get(onboardingURL)
	if err != nil {
		return "", err
	}

	var tmpPath string
	err = json.NewDecoder(res.Body).Decode(&tmpPath)
	if err != nil {
		return "", err
	}

	// Copy file to directory accessible by JupyterLab
	base := filepath.Base(tmpPath)
	newPath := filepath.Join(params.ServerRoot, base)
	_, err = copyFile(tmpPath, newPath)
	if err != nil {
		return "", err
	}

	return newPath, nil
}
