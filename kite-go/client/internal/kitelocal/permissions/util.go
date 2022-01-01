package permissions

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang"
)

// IsSupportedLangExtension returns whether the path is a suported extension of the languages
func IsSupportedLangExtension(path string, langs map[lang.Language]struct{}) (bool, error) {
	if path == "" {
		return false, nil
	}

	var supported bool
	pathLang := lang.FromFilename(path)
	for lang := range langs {
		if pathLang == lang {
			supported = true
		}
	}

	return supported, nil
}

// Filename extracts a filename from the request, and resets the request body
func filename(r *http.Request) (fn string) {
	if fn := mux.Vars(r)["filename"]; fn != "" {
		return fn
	}
	if fn := r.URL.Query().Get("filename"); fn != "" {
		return fn
	}

	req := make(map[string]interface{})
	buf, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewReader(buf))
	if err := json.NewDecoder(bytes.NewReader(buf)).Decode(&req); err != nil {
		return ""
	}

	val, exists := req["filename"]
	if !exists {
		return ""
	}
	if fn, ok := val.(string); ok {
		return fn
	}
	return ""
}
