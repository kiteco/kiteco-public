//go:generate goversioninfo -64 -o resource_windows.syso -icon app.ico

package main

//#cgo windows LDFLAGS: -L${SRCDIR}/../../../../windows/tensorflow/lib
//#cgo windows LDFLAGS: -static-libgcc -static-libstdc++ -Wl,-Bstatic -lstdc++ -lpthread -Wl,-Bdynamic
//#cgo linux LDFLAGS: -Wl,-rpath,$ORIGIN
import "C"

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/kiteco/kiteco/kite-go/client/internal/client"
	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

func fatal(err error) {
	switch err {
	// ErrRunning occurs when kite's port was already in use
	// but kite was manually launched. In this case, we just launch
	// the sidebar. We shouldn't rollbar or alert anything.
	case clientapp.ErrRunning:
	default:
		rollbar.Critical(err)
		clientapp.Alert(err)
	}

	log.Println(err)
	os.Exit(1)
}

// todo(jansorg): Cleanup temp dirs root-dir and index-dir?
func main() {
	rollbar.Disable()
	clienttelemetry.Disable()

	// custom flags to support integration testing
	var port int
	var rootDir string
	var indexedDir string
	var dummyUser bool
	var eventRequestDelay time.Duration
	var requestDelay time.Duration

	// we use our own set of flags because the testing package added its own flags to the global set.
	// these would show up in the output of "--help".
	// the testing package is initialized because we share test code between packages, e.g. in component/testing.go.
	var commandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	commandLine.IntVar(&port, "port", 56624, "The port where kited will be listenening")
	commandLine.StringVar(&rootDir, "root-dir", "", "The directory where kited stores its configuration data. The equivalent of ~/.kite. Defaults to a new temporary directory.")
	commandLine.StringVar(&indexedDir, "index-dir", "", "The directory indexed by kited. Defaults to a new temporary directory.")
	commandLine.BoolVar(&dummyUser, "dummy-user", true, "If 'true' then a dummy user will be authenticated")
	commandLine.DurationVar(&requestDelay, "request-delay", 0, "Artificial delay for all requests.")
	commandLine.DurationVar(&eventRequestDelay, "event-request-delay", 0, "Artificial delay for requests to /clientapi/editor/event. This overrides the generic request delay.")
	commandLine.Parse(os.Args[1:])

	if rootDir == "" {
		rootDir = newTempDir("kite-testing-root")
	}

	if indexedDir == "" {
		indexedDir = newTempDir("kite-testing-indexed")
	}

	if dummyUser {
		// write a dummy user into ROOT/user to make the auth component use it in offline mode
		buf, err := json.Marshal(&community.User{
			ID:            -1,
			Name:          "Dummy User",
			Email:         "kite-int@kite.com",
			EmailVerified: true,
		})
		if err != nil {
			fatal(fmt.Errorf("error encoding for user caching: %v", err))
		}
		if err = ioutil.WriteFile(filepath.Join(rootDir, "user"), buf, os.ModePerm); err != nil {
			fatal(fmt.Errorf("error caching user to file: %v", err))
		}

		// save a dummy auth cookie at ROOT/session.json to force the main loop load to load the user from the auth component
		buf, err = json.Marshal(&[]http.Cookie{
			{Name: "kite-session", Value: "value", Path: "", Domain: ""},
		})
		if err != nil {
			fatal(fmt.Errorf("error marshalling auth: %v", err))
		}
		if err = ioutil.WriteFile(filepath.Join(rootDir, "session.json"), buf, os.ModePerm); err != nil {
			fatal(fmt.Errorf("error writing into session.jsoN: %v", err))
		}
	}

	// recorder of HTTP calls which also is a component and registers HTTP endpoints at /test/
	history := newCallHistory()

	// use a offline network manager to disable the remote user check in the auth component
	networkMgr := &mockserver.MockNetworkManager{}
	networkMgr.SetOnline(false)

	// request handler to add the configured request delay, must be added after history.recordCall to be wrapped
	delayHandler := func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if strings.HasPrefix(r.URL.Path, "/testapi/") {
			next.ServeHTTP(rw, r)
			return
		}

		delay := requestDelay
		if r.URL.Path == "/clientapi/editor/event" {
			delay = eventRequestDelay
		}

		time.Sleep(delay)
		next.ServeHTTP(rw, r)
	}

	opts := client.Options{
		TestRootDir: rootDir,
		LocalOpts: kitelocal.Options{
			IndexedDir: indexedDir,
		},
		Network: networkMgr,
		TestHandlers: func(handlers []negroni.Handler) []negroni.Handler {
			// recordCall is first, then the default handlers and delayHandler at last
			// delayHandlers has to be last because one of the default handlers logs the response time
			newHandlers := []negroni.Handler{negroni.HandlerFunc(history.recordCall)}
			newHandlers = append(newHandlers, handlers...)
			newHandlers = append(newHandlers, negroni.HandlerFunc(delayHandler))
			return newHandlers
		},
		Updater: updates.NewMockManager(),
	}

	c, _, err := clientapp.StartPort(context.Background(), port, true, &opts, history)
	if err != nil {
		fatal(err)
	}

	err = c.Connect(c.Settings.Server())
	if err != nil {
		fatal(err)
	}

	// sleep forever
	select {}
}

func newTempDir(suffix string) string {
	dir, err := ioutil.TempDir("", suffix)
	if err != nil {
		fatal(err)
	}
	return dir
}
