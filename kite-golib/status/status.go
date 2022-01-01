//go:generate go-bindata -pkg status -o bindata.go templates

package status

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	humanize "github.com/dustin/go-humanize"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-golib/gziphttp"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

func init() {
	http.HandleFunc("/debug/status", handler)
	http.HandleFunc("/debug/status-json", gziphttp.Wrap(handlerJSON))

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates = templateset.NewSet(staticfs, "templates", template.FuncMap{
		"sectionID": func(val string) string {
			return strings.Replace(strings.Replace(val, " ", "_", -1), "/", "-", -1)
		},
		"humanizeSize": func(val int64) string {
			return humanize.Bytes(uint64(val))
		},
		"humanizeDuration": func(val interface{}) string {
			switch t := val.(type) {
			case int64:
				return time.Duration(t).String()
			case float64:
				return time.Duration(int64(t)).String()
			}
			return "n/a"
		},
	})

	if err := templates.Validate(); err != nil {
		log.Fatalln("template validation error:", err)
	}
}

var (
	templates *templateset.Set
	s         = newEmptyStatus()
)

// Status is the root level object containing all sections.
type Status struct {
	m        sync.Mutex
	Sections map[string]*Section
}

func newEmptyStatus() *Status {
	return &Status{
		Sections: make(map[string]*Section),
	}
}

// MarshalJSON allows for go-routine safe access to Sections.
func (s *Status) MarshalJSON() ([]byte, error) {
	s.m.Lock()
	defer s.m.Unlock()

	// to avoid recursive call into MarshalJSON (and the subsequent deadlock),
	// create a temporary type to mask the MarshalJSON method
	type tmp Status
	return json.Marshal((*tmp)(s))
}

func (s *Status) aggregate(other *Status) {
	for key, section := range other.Sections {
		mine, exists := s.Sections[key]
		if !exists {
			mine = newEmptySection(key)
			s.Sections[key] = mine
		}
		mine.aggregate(section)
	}
}

// NOTE: extra space before section name is intentional. it there to ensure
// this section sorts above any other section when added to the Sections map.
const headlinesHeader = " Headlines"

func (s *Status) updateHeadlines() {
	headlines := newEmptySection(headlinesHeader)

	for _, section := range s.Sections {
		for key, counter := range section.Counters {
			if counter.Headline {
				headlines.Counters[key] = counter
			}
		}

		for key, ratio := range section.Ratios {
			if ratio.Headline {
				headlines.Ratios[key] = ratio
			}
		}

		for key, breakdown := range section.Breakdowns {
			if breakdown.Headline {
				headlines.Breakdowns[key] = breakdown
			}
		}

		for key, sample := range section.SampleInt64s {
			if sample.Headline {
				headlines.SampleInt64s[key] = sample
			}
		}

		for key, sample := range section.SampleBytes {
			if sample.Headline {
				headlines.SampleBytes[key] = sample
			}
		}

		for key, sample := range section.SampleDurations {
			if sample.Headline {
				headlines.SampleDurations[key] = sample
			}
		}

		for key, counter := range section.CounterDistributions {
			if counter.Headline {
				headlines.CounterDistributions[key] = counter
			}
		}

		for key, counter := range section.RatioDistributions {
			if counter.Headline {
				headlines.RatioDistributions[key] = counter
			}
		}

		for key, counter := range section.BoolDistributions {
			if counter.Headline {
				headlines.BoolDistributions[key] = counter
			}
		}

		for key, counter := range section.DurationDistributions {
			if counter.Headline {
				headlines.DurationDistributions[key] = counter
			}
		}
	}

	// Check if there are any headlines
	if len(headlines.Counters)+
		len(headlines.Ratios)+
		len(headlines.Breakdowns)+
		len(headlines.SampleInt64s)+
		len(headlines.SampleDurations)+
		len(headlines.SampleBytes)+
		len(headlines.CounterDistributions)+
		len(headlines.BoolDistributions)+
		len(headlines.DurationDistributions) > 0 {

		// Add headline section if there were headline metrics found
		s.Sections[headlinesHeader] = headlines
	} else {
		// Delete headline section otherwise
		delete(s.Sections, headlinesHeader)
	}
}

// ShallowCopy only copies the Sections field (and not the mutex)
func (s *Status) ShallowCopy(status *Status) {
	s.Sections = status.Sections
}

// --

// Get returns the *Status object
func Get() *Status {
	return s
}

// Poll the provided hostPort for its status
func Poll(hostPort *url.URL) (*Status, error) {
	pollURL, err := hostPort.Parse("/debug/status-json")
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout:   time.Second * 15,
		Transport: http.DefaultTransport,
	}

	resp, err := client.Get(pollURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type tmp struct {
		Status *Status `json:"status"`
	}

	var tmpStatus tmp
	if err := json.NewDecoder(resp.Body).Decode(&tmpStatus); err != nil {
		return nil, err
	}

	if tmpStatus.Status == nil {
		return nil, fmt.Errorf("no status found")
	}

	return tmpStatus.Status, nil
}

// Aggregate multiple status objects
func Aggregate(statuses []*Status) *Status {
	agg := newEmptyStatus()
	for _, status := range statuses {
		agg.aggregate(status)
	}
	return agg
}

// Render will render the status page using the provided status object
func Render(release string, status *Status, w http.ResponseWriter, r *http.Request) {
	type templateData struct {
		Release string
		Status  *Status
	}

	status.updateHeadlines()

	data := templateData{
		Release: release,
		Status:  status,
	}

	if err := templates.Render(w, "index.html", data); err != nil {
		log.Println("render error:", err)
	}
}

// RenderEmbedded will render the status page using the provided status object
func RenderEmbedded(title string, status *Status, w io.Writer) error {
	type templateData struct {
		Title  string
		Status *Status
	}

	data := templateData{
		Title:  title,
		Status: status,
	}

	return templates.Render(w, "embedded.html", data)
}

// --

func handler(w http.ResponseWriter, r *http.Request) {
	Render(os.Getenv("RELEASE"), s, w, r)
}

func handlerJSON(w http.ResponseWriter, r *http.Request) {
	type statusResponse struct {
		Status *Status `json:"status"`
	}
	resp := statusResponse{Status: s}
	json.NewEncoder(w).Encode(&resp)
}
