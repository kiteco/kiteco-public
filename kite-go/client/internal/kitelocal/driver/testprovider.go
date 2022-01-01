package driver

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"sync"

	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-go/diff"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// TestDriverGenerator allows customization of the drivers created by the TestProvider
type TestDriverGenerator func(filename, editor, content string) (core.FileDriver, http.Handler)

// TestProvider is a implementation of the Provider interface used for testing
type TestProvider struct {
	m         sync.Mutex
	drivers   map[driverKey]*State
	generator TestDriverGenerator
}

// NewTestProvider ...
func NewTestProvider() *TestProvider {
	return &TestProvider{
		drivers:   make(map[driverKey]*State),
		generator: defaultGenerator,
	}
}

// TestDriverGenerator allows for custom driver generation from the Provider
func (t *TestProvider) TestDriverGenerator(gen TestDriverGenerator) {
	t.generator = gen
}

// Driver implements Provider
func (t *TestProvider) Driver(ctx kitectx.Context, filename, editor, state string) (*State, bool) {
	ctx.CheckAbort()

	t.m.Lock()
	defer t.m.Unlock()
	key := driverKey{filename, editor, state}
	s, ok := t.drivers[key]
	return s, ok
}

// DriverFromContent implements Provider
func (t *TestProvider) DriverFromContent(ctx kitectx.Context, filename, editor, content string, cursor int) *State {
	ctx.CheckAbort()

	t.m.Lock()
	defer t.m.Unlock()

	state := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	key := driverKey{filename, editor, state}
	if s, ok := t.drivers[key]; ok {
		return s
	}

	driver, handler := t.generator(filename, editor, content)
	s := &State{
		Filename:      filename,
		Editor:        editor,
		State:         state,
		FileDriver:    driver,
		BufferHandler: handler,
	}

	t.drivers[key] = s

	return s
}

// LatestDriver returns the latest available file driver of the file
func (t *TestProvider) LatestDriver(ctx kitectx.Context, unixFilepath string) *python.UnifiedDriver {
	return nil
}

// --

type driverKey struct {
	filename string
	editor   string
	state    string
}

func defaultGenerator(filename, editor, state string) (core.FileDriver, http.Handler) {
	driver := diff.NewBufferDriver()
	return driver, &defaultHandler{}
}

type defaultHandler struct{}

func (d *defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
