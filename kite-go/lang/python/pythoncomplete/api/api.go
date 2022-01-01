package api

import (
	"context"
	"log"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/driver"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/local-pipelines/mixing"
)

const maxEditorEvents = 20

// LocalContext is a subtype of localcode.Context
type LocalContext interface {
	RequestForFile(filename string, trackExtra interface{}) (interface{}, error)
}

// Options bundles API options.
// If a ResourceManager is not provided, one will be loaded synchronously.
// If Models are not provided, they are not loaded, and model-dependent functionality is not provided.
type Options struct {
	ResourceManager           pythonresource.Manager
	Models                    *pythonmodels.Models
	LexicalModels             *lexicalmodels.Models
	LocalContext              LocalContext
	GGNNSubtokenEnabled       bool
	GGNNSubtokenEnabledByFlag bool

	Cloud bool
}

// API manages per-file completions state and exports API
type API struct {
	opts       Options
	drivers    umfDrivers
	events     *api.Events
	normalizer mixing.Normalizer
	product    licensing.ProductGetter

	lastUsedLock *sync.Mutex
	lastUsed     *time.Time
	unloadTimer  *time.Timer
}

// New allocates a new API. It may panic if a resource manager is not provided and fails to load or if loading the normalizer fails.
func New(ctx context.Context, opts Options, pa licensing.ProductGetter) API {
	if opts.ResourceManager == nil {
		rm, errc := pythonresource.NewManagerWithCtx(ctx, pythonresource.DefaultLocalOptions)
		if err := <-errc; err != nil {
			panic(err)
		}
		opts.ResourceManager = rm
	}

	now := time.Now()
	a := API{
		opts:         opts,
		product:      pa,
		lastUsedLock: &sync.Mutex{},
		lastUsed:     &now,
	}

	unloadInterval := 15 * time.Minute
	go func() {
		ticker := time.NewTicker(unloadInterval / 2)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				func() {
					a.lastUsedLock.Lock()
					defer a.lastUsedLock.Unlock()

					if a.lastUsed != nil && !a.lastUsed.IsZero() && time.Since(*a.lastUsed) >= unloadInterval {
						log.Println("Unloading data for python")
						opts.Models.Reset()
						a.drivers.Reset()
						*a.lastUsed = time.Time{}
					}
				}()
			}
		}
	}()

	if opts.Cloud {
		// no cap on # of drivers, but expire them after 5 seconds.
		a.drivers = newUMFDrivers(0, 5*time.Second)
	} else {
		// at most one driver at a time, but don't expire it.
		a.drivers = newUMFDrivers(1, 0)
		// assume single user in non-Cloud-land
		// TODO(naman) figure out what to do for multi-user deployments
		a.events = api.NewEvents(20)
	}

	normalizer, err := mixing.NewNormalizer()
	if err != nil {
		panic(err)
	}
	a.normalizer = normalizer
	return a
}

// Complete provides completions with the given options
func (a API) Complete(ctx kitectx.Context, opts CompleteOptions, req data.APIRequest, pyctx *python.Context, metricFn data.EngineMetricsCallback) data.APIResponse {
	return a.update(ctx, opts, req, pyctx, true, metricFn)
}

// Edit triggers the completions engine on the edit
func (a API) Edit(ctx kitectx.Context, opts CompleteOptions, req data.APIRequest, pyctx *python.Context) {
	resp := a.update(ctx, opts, req, pyctx, false, nil)
	if resp.Error != "" {
		log.Println(resp.Error)
	}
}

func (a API) update(ctx kitectx.Context, opts CompleteOptions, req data.APIRequest, pyctx *python.Context, requestCompletions bool, metricFn data.EngineMetricsCallback) data.APIResponse {
	defer func() {
		a.lastUsedLock.Lock()
		defer a.lastUsedLock.Unlock()

		*a.lastUsed = time.Now()
	}()

	resp := data.NewAPIResponse(req)

	if !path.IsAbs(req.UMF.Filename) {
		err := errors.Errorf("complete request must contain absolute path")
		rollbar.Error(err, req.Filename, req.Editor)
		resp.HTTPStatus = http.StatusBadRequest
		resp.Error = err.Error()
		return resp
	}

	if a.opts.GGNNSubtokenEnabled {
		opts.MixOptions.GGNNSubtokenEnabled = true
		opts.ScheduleOptions.GGNNSubtokenEnabled = true
	}
	if req.Editor == data.AtomEditor && !a.opts.GGNNSubtokenEnabledByFlag {
		opts.MixOptions.GGNNSubtokenEnabled = false
		opts.ScheduleOptions.GGNNSubtokenEnabled = false
	}

	var idx *pythonlocal.SymbolIndex
	if a.opts.LocalContext != nil {
		obj, err := a.opts.LocalContext.RequestForFile(req.Filename, nil)
		if err == nil {
			idx = obj.(*pythonlocal.SymbolIndex)
		}
	}
	global := pythonproviders.Global{
		ResourceManager: a.opts.ResourceManager,
		Models:          a.opts.Models,
		FilePath:        req.Filename,
		LocalIndex:      idx,
		Product:         a.product,
		Lexical: lexicalproviders.Global{
			Models:   a.opts.LexicalModels,
			FilePath: req.Filename,
			Product:  a.product,
		},
		Normalizer: a.normalizer,
	}

	if opts.MixOptions.RenderOptions.SingleTokenProCompletion {
		// the product is Pro for the purposes of generating completions
		global.Product = licensing.Pro
		global.Lexical.Product = licensing.Pro
	}

	// The pythoncomplete provider will see global.Product == licensing.Pro
	// and continue to generate completions. This provider then dispatches to
	// the lexicalcomplete Python provider, which sees global.Lexical.Product ==
	// licensing.Free. Since lexical Python completions are Pro-only,
	// the lexicalcomplete provider aborts.
	if opts.MixOptions.LexicalCompletionsDisabled {
		global.Lexical.Product = licensing.Free
	}

	global.UserID = req.UserID
	global.MachineID = req.MachineID
	global.Lexical.UserID = req.UserID
	global.Lexical.MachineID = req.MachineID
	if a.events != nil {
		global.Lexical.EditorEvents = a.events.Collect()
	}

	d := a.drivers.Get(req.UMF)
	compls, err := d.Update(ctx, opts, global, req.SelectedBuffer, pyctx, requestCompletions, metricFn)
	if err != nil {
		resp.Error = err.Error()
		resp.HTTPStatus = http.StatusInternalServerError
		return resp
	}
	resp.Completions = compls
	return resp
}

// Reset clears state
func (a API) Reset() {
	a.drivers.Reset()
}

// CreateSchedulerFixture gets the driver for the given key and returns a
// test fixture from its scheduler's cache.
func (a API) CreateSchedulerFixture(k data.UMF) (driver.Fixture, error) {
	d, err := a.drivers.GetNoUpdate(k)
	if err != nil {
		return driver.Fixture{}, err
	}
	return d.CreateSchedulerFixture(), nil
}

// PushEditorEvent stores the event
func (a API) PushEditorEvent(evt *component.EditorEvent) {
	if a.events != nil {
		a.events.Push(evt)
	}
}
