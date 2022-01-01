package midware

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// Wrap wraps the provided http.ServeMux with the default set of middleware.
func Wrap(handler http.Handler) http.Handler {
	logger := contextutil.BasicLogger()
	if handler == nil {
		handler = http.DefaultServeMux
	}
	return negroni.New(
		NewRecovery(),
		NewLogger(logger),
		negroni.Wrap(handler),
	)
}

// Logger is a HTTP request logger for use as negroni middleware.
type Logger struct {
	logger *log.Logger
}

// NewLogger returns a Logger negroni.Handler that will log requests
// to the provided logger.
func NewLogger(logger *log.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

// ServeHTTP implements negroni.Handler
func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	next(w, r)
	url := r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.Query().Encode()
	}

	user := community.GetUser(r)
	machine := community.GetMachine(r)
	if machine == "" {
		machine = "(no machine set)"
	}

	if ignoreForLogging(r.Method, url) {
		return
	}

	switch rw := w.(type) {
	case negroni.ResponseWriter:
		// Log HTTP status and content size if this is a negroni.ResponseWriter
		if user != nil {
			l.logger.Println(r.Method, url, rw.Status(), rw.Size(), time.Now().Sub(start), user.ID, user.Email, machine)
		} else {
			l.logger.Println(r.Method, url, rw.Status(), rw.Size(), time.Now().Sub(start))
		}

	case http.ResponseWriter:
		// Log what we can for http.ResponseWriter
		if user != nil {
			l.logger.Println(r.Method, url, time.Now().Sub(start), user.ID, user.Email, machine)
		} else {
			l.logger.Println(r.Method, url, time.Now().Sub(start))
		}
	}
}

var ignorePatterns = []*regexp.Regexp{
	regexp.MustCompile("GET /clientapi/settings/*"),
	regexp.MustCompile("GET /clientapi/online"),
}

func ignoreForLogging(method string, url string) bool {
	s := method + " " + url
	for _, re := range ignorePatterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// --

// Recovery is a panic recovery middleware handler for negroni.
type Recovery struct {
	PrintStack bool
	StackAll   bool
	StackSize  int
}

// NewRecovery returns a new Recovery negroni.Handler
func NewRecovery() *Recovery {
	return &Recovery{
		PrintStack: true,
		StackAll:   true,
		StackSize:  1028 * 8,
	}
}

// ServeHTTP implements negroni.Handler
func (rec *Recovery) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	defer func(req *http.Request) {
		if err := recover(); err != nil {
			// Don't write to the response writer if this is a websocket
			// The connection has already been hijacked, making this write invalid.
			switch r.URL.Scheme {
			case "http", "https":
				// Return a 500 InternalServerError
				w.WriteHeader(http.StatusInternalServerError)
			}

			// Grab the stack trace
			stack := make([]byte, rec.StackSize)
			stack = stack[:runtime.Stack(stack, rec.StackAll)]
			logger := contextutil.BasicLogger()
			logger.Println("[recovery!]", req.Method, req.URL.Path, fmt.Sprintf("PANIC: %s\n%s", err, stack))

			// Report this error to Rollbar
			rollbar.RecoveryError(fmt.Errorf("%s", err), req)
		}
	}(r)

	next(w, r)
}

// NoCache is a middleware handler for setting no-cache headers.
type NoCache struct{}

// NewNoCache returns a NoCache negroni.Handler that sets the no-cache headers.
func NewNoCache() *NoCache {
	return &NoCache{}
}

// ServeHTTP implements negroni.Handler
func (nc *NoCache) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	newRw := negroni.NewResponseWriter(w)

	// ensure no caching occurs, must occur before response has been written
	newRw.Before(func(rw negroni.ResponseWriter) {
		h := rw.Header()
		h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		h.Set("Pragma", "no-cache")
		h.Set("Expires", "0")
	})

	next(newRw, r)
}
