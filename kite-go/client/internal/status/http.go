package status

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/navigation/codebase"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	constants "github.com/kiteco/kiteco/kite-golib/conversion"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/enginestatus"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/presentation"
)

func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	var response enginestatus.Response

	filetype := r.URL.Query().Get("filetype")
	filename := r.URL.Query().Get("filename")
	// checkloaded is used for testing status without waiting on model warmup
	ckloaded := r.URL.Query().Get("checkloaded") != "false"

	switch {
	case filename == "" && filetype == "":
		fallthrough
	case filename != "" && filetype != "":
		http.Error(w, "exactly one of the filename & filetype parameters must be specified", http.StatusBadRequest)
		return

	case filetype != "":
		if filetype == "python" {
			response.Status = "noIndex"
			response.Short = "ready (no index)"
			response.Long = "Kite can only index your code once it's saved"
		} else {
			response.Status = "unsupported"
			response.Short = "unsupported"
			response.Long = "Kite currently only supports Python"
		}

	default: // filename != ""
		canonfn, err := canonicalPath(filename)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid path: %v", err), http.StatusBadRequest)
		}
		response = m.statusReponse(filename, canonfn, ckloaded)
		response = m.maybeAppendCRODetails(response)
	}

	buf, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("error encoding json response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Write(buf)
}

func (m *Manager) statusReponse(filename, canonical string, checkLoaded bool) enginestatus.Response {
	ss := m.permissions.IsSupportedExtension(filename)
	if !ss.CompletionsSupported {
		if applesilicon.Detected {
			return enginestatus.Response{
				Status: "unsupported",
				Short:  "unsupported",
				Long:   "Kite only supports Python for Apple Silicon",
			}
		}
		return enginestatus.Response{
			Status: "unsupported",
			Short:  "unsupported",
			Long:   "Kite doesn't support the current file type",
		}
	}

	naverr := m.navValidate(filename)
	fext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if checkLoaded && !m.isLoaded(fext) && !applesilicon.Detected {
		return enginestatus.Response{
			Status: "initializing",
			Short:  "initializing",
			Long:   "Kite is warming up",
		}
	} else if naverr == codebase.ErrProjectStillIndexing {
		return enginestatus.Response{
			Status: "indexing",
			Short:  "indexing",
			Long:   "Kite is indexing your code, but should still function in the meanwhile.",
		}
	} else if fext == "py" {
		// Indexing is only relevant in the Python use case.
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return enginestatus.Response{
				Status: "noIndex",
				Short:  "ready (no index)",
				Long:   "Kite can only index your code once it's saved",
			}
		}

		if !m.fileIndexed(canonical) {
			return enginestatus.Response{
				Status: "indexing",
				Short:  "indexing",
				Long:   "Kite is indexing your code, but should still function in the meanwhile.",
			}
		}
	}

	return enginestatus.Response{
		Status: "ready",
		Short:  "ready",
		Long:   "Kite is ready!",
	}
}

// appends number of completions left for usage-paywall users with all_features_pro flag set
func (m *Manager) maybeAppendCRODetails(resp enginestatus.Response) enginestatus.Response {
	pluralize := func(s string, n int) string {
		if n != 1 {
			return s + "s"
		}
		return s
	}

	if _, _, plan, _ := m.license.LicenseStatus(); plan.IsSubscriber() {
		return resp
	}

	if m.cohort.ConversionCohort() == constants.UsagePaywall {
		if allpro, _ := m.settings.GetBool(settings.AllFeaturesPro); allpro {
			remaining, err := m.settings.GetInt(settings.PaywallCompletionsRemaining)
			if err != nil {
				return resp
			}
			if remaining > 0 {
				return enginestatus.Response{
					Status: fmt.Sprintf("%s (%d %s left today)", resp.Status, remaining, pluralize("completion", remaining)),
					Short:  fmt.Sprintf("%s (%d %s left today)", resp.Short, remaining, pluralize("completion", remaining)),
					Long:   fmt.Sprintf("%s (%d %s left today)", resp.Long, remaining, pluralize("completion", remaining)),
				}
			}
			return enginestatus.Response{
				Status: "locked (upgrade to Pro to unlock)",
				Short:  "locked (upgrade to Pro to unlock)",
				Long:   "Locked until tomorrow. Upgrade now to unlock.",
				Button: &presentation.Button{
					Text:   "Upgrade to Pro",
					Action: presentation.Open,
					Link:   proto.String(fmt.Sprintf("https://%s/pro", domains.WWW)),
				},
			}
		}
	}
	return resp
}

func (m *Manager) fileIndexed(filename string) bool {
	// Need lock here to access localCodeStatus field.
	m.mu.Lock()
	defer m.mu.Unlock()

	fp := spooky.Hash64([]byte(filename))
	for _, s := range m.localCodeStatus.Indices {
		// In Kite Local, we can check against the file hashes in the index
		if _, ok := s.FileHashes[fp]; ok {
			return true
		}
	}
	return false
}

func canonicalPath(filename string) (string, error) {
	// convert paths to unix and lowercase for windows
	// to ensure case consistency and to match the path
	// specified in the index.
	canonical, err := localpath.ToUnix(filename)
	if err != nil {
		return "", errors.Errorf("invalid path: %v", err)
	}
	if runtime.GOOS == "windows" {
		canonical = strings.ToLower(canonical)
	}
	return canonical, nil
}
