package component

import (
	"net/http"

	"github.com/kiteco/kiteco/kite-go/lang"
)

// PermissionsManager defines the functions to work with permission data.
// Component interfaces must not depend on implementations
type PermissionsManager interface {
	Core

	// Filename extracts the filename from the incoming request, and resets the body if necessary
	Filename(r *http.Request) string

	// WrapAuthorizedFile returns a new http HandlerFunc which first checks for authorized
	// before passing control to the original handler
	WrapAuthorizedFile(handler http.HandlerFunc) http.HandlerFunc

	// IsSupportedExtension returns whether the path is a suported extension
	IsSupportedExtension(path string) SupportStatus

	// IsSupportedLangExtension returns whether the path is a supported extension of the languages
	IsSupportedLangExtension(path string, langs map[lang.Language]struct{}) (bool, error)
}

// SupportStatus encapsulates information on which features are supported
type SupportStatus struct {
	EditEventSupported   bool `json:"editEventSupported"`
	CompletionsSupported bool `json:"completionsSupported"`
	HoverSupported       bool `json:"hoverSupported"`
	SignaturesSupported  bool `json:"signaturesSupported"`
}
