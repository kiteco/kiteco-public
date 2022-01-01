package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/exec"
)

func (c *Client) handleDefaultEmail(w http.ResponseWriter, r *http.Request) {
	email, message := getDefaultEmail()
	data, err := json.Marshal(map[string]interface{}{
		"email":   email,
		"message": message,
	})
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func getDefaultEmail() (string, string) {
	// git config --global user.email
	cmd := exec.Command("git", "config", "--global", "user.email")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return "", ""
	}
	if err := cmd.Wait(); err != nil {
		return "", ""
	}
	email := strings.TrimSpace(out.String())

	if strings.HasSuffix(strings.ToLower(email), "@users.noreply.github.com") {
		return "", ""
	}
	return email, "Using your Git config email"
}
