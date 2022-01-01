package websandbox

import (
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/pkg/errors"
)

// LoadServices creates and returns a python.Services objext according to the input
// serviceOptions for a websandbox environment
func LoadServices(serviceOptions *python.ServiceOptions) (*python.Services, error) {
	if serviceOptions == nil {
		serviceOptions = &python.DefaultServiceOptions
	}
	serviceOptions.ModelOptions.Local = false
	// Tweak resouce manager options as needed
	var dists []keytypes.Distribution
	rmOpts := pythonresource.DefaultLocalOptions
	rmOpts.Concurrency = 32

	rmOpts.Dists = dists
	resourceManager, errc := pythonresource.NewManager(rmOpts)
	if err := <-errc; err != nil {
		return nil, err
	}

	importGraph := pythonimports.NewEmptyGraph()

	models, err := pythonmodels.New(serviceOptions.ModelOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create models")
	}

	return &python.Services{
		Options:         serviceOptions,
		ResourceManager: resourceManager,
		// a lot of places don't check nil, so just use an empty client
		// TODO remove soon once we get the resource manager version working
		ImportGraph: importGraph,
		Models:      models,
	}, nil
}

// CompletionRequest is the expected shape of the Body of a request to the completions endpoint
type CompletionRequest struct {
	Text        string `json:"text"`
	CursorBytes int64  `json:"cursor_bytes"`
	CursorRunes int64  `json:"cursor_runes"`
	Filename    string `json:"filename"`
	ID          string `json:"id"`
}

// validate returns an error if the request is malformed
func (c CompletionRequest) validateCompletions() error {
	_, err := webutils.OffsetToBytes([]byte(c.Text), int(c.CursorBytes), int(c.CursorRunes))
	if err != nil {
		return err
	}

	return nil
}

// cursor returns the cursor offset in bytes, as inferred from CursorBytes and
// CursorRunes. It is assumed to be called on valid requests only.
func (c CompletionRequest) cursor() int64 {
	cursor, _ := webutils.OffsetToBytes([]byte(c.Text), int(c.CursorBytes), int(c.CursorRunes))
	return int64(cursor)
}
