package midware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/codegangsta/negroni"
	"github.com/kiteco/kiteco/kite-golib/status"
)

var (
	section          = status.NewSection("Response Codes")
	editorAPI        = section.Breakdown("Editor API")
	syncAPI          = section.Breakdown("File Sync API")
	loginAPI         = section.Breakdown("Login API")
	downloadAPI      = section.Breakdown("Download API")
	winUpdateAPI     = section.Breakdown("Win Update API")
	macUpdateAPI     = section.Breakdown("Mac Update API")
	localCodeAPI     = section.Breakdown("Local Code Worker API")
	clientLogsAPI    = section.Breakdown("Client Logs API")
	windowsCrashAPI  = section.Breakdown("Windows Crash API")
	serviceStatusAPI = section.Breakdown("Service Status API")
)

func init() {
	editorAPI.Headline = true
	syncAPI.Headline = true
	loginAPI.Headline = true
	downloadAPI.Headline = true
	winUpdateAPI.Headline = true
	macUpdateAPI.Headline = true
	localCodeAPI.Headline = true
	clientLogsAPI.Headline = true
	windowsCrashAPI.Headline = true
	serviceStatusAPI.Headline = true
}

// StatusResponseCodes tracks response codes for API's we want to track
type StatusResponseCodes struct{}

// ServeHTTP implements negroni.Handler
func (s *StatusResponseCodes) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	next(w, r)

	path := r.URL.Path
	var breakdown *status.Breakdown

	switch {
	case strings.HasPrefix(path, "/api/editor/") || strings.HasPrefix(path, "/api/buffer/"):
		breakdown = editorAPI
	case strings.HasPrefix(path, "/local/"):
		breakdown = syncAPI
	case path == "/api/account/login-web" || path == "/api/account/login-desktop":
		breakdown = loginAPI
	case strings.HasPrefix(path, "/release/dls/"):
		breakdown = downloadAPI
	case path == "/release/appcast.xml":
		breakdown = macUpdateAPI
	case strings.HasPrefix(path, "/release/windows/"):
		breakdown = winUpdateAPI
	case strings.HasPrefix(path, "/artifacts/"):
		breakdown = localCodeAPI
	case strings.HasPrefix(path, "/clientlogs"):
		breakdown = clientLogsAPI
	case strings.HasPrefix(path, "/windowscrash"):
		breakdown = windowsCrashAPI
	case strings.HasPrefix(path, "/servicestatus"):
		breakdown = serviceStatusAPI
	}

	if breakdown != nil {
		nw := w.(negroni.ResponseWriter)
		breakdown.HitAndAdd(fmt.Sprintf("%d", nw.Status()))
	}
}
