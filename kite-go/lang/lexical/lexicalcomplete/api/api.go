package api

import (
	"context"
	"log"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const maxEditorEvents = 20

// Options bundles API options.
type Options struct {
	Models *lexicalmodels.Models
	Cloud  bool
}

// API manages per-file completions state and exports API
type API struct {
	opts    Options
	drivers umfDrivers
	events  *Events
	product licensing.ProductGetter

	lastUsedLock *sync.Mutex
	lastUsed     map[lang.Language]time.Time
	unloadTimer  *time.Timer
}

// New allocates a new API. It may panic if a resource manager is not provided and fails to load.
func New(ctx context.Context, opts Options, pa licensing.ProductGetter) API {
	a := API{
		opts:         opts,
		product:      pa,
		lastUsedLock: &sync.Mutex{},
		lastUsed:     make(map[lang.Language]time.Time),
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

					for language, lastUsed := range a.lastUsed {
						if time.Since(lastUsed) >= unloadInterval {
							log.Println("Unloading data for language", language.Name())

							lexicalproviders.ResetLanguage(language)
							opts.Models.ResetLanguage(language)
							a.drivers.Reset()

							delete(a.lastUsed, language)
						}
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
		a.events = NewEvents(20)
	}
	return a
}

// Complete provides completions with the given options
func (a API) Complete(ctx kitectx.Context, opts CompleteOptions, req data.APIRequest, fd core.FileDriver, metricFn data.EngineMetricsCallback) data.APIResponse {
	return a.update(ctx, opts, req, fd, true, metricFn)
}

// Edit triggers the completions engine on the edit
func (a API) Edit(ctx kitectx.Context, opts CompleteOptions, req data.APIRequest, fd core.FileDriver) {
	resp := a.update(ctx, opts, req, fd, false, nil)
	if resp.Error != "" {
		log.Println(resp.Error)
	}
}

func (a API) update(ctx kitectx.Context, opts CompleteOptions, req data.APIRequest, fd core.FileDriver, requestCompletions bool, metricFn data.EngineMetricsCallback) data.APIResponse {
	resp := data.NewAPIResponse(req)
	defer CompletionDuration.DeferRecord(time.Now())
	if !path.IsAbs(req.UMF.Filename) {
		err := errors.Errorf("complete request must contain absolute path")
		rollbar.Error(err, req.Filename, req.Editor)
		resp.HTTPStatus = http.StatusBadRequest
		resp.Error = err.Error()
		return resp
	}

	global := lexicalproviders.Global{
		Models:   a.opts.Models,
		FilePath: req.Filename,
		Product:  a.product,
	}
	global.UserID = req.UserID
	global.MachineID = req.MachineID
	if a.events != nil {
		global.EditorEvents = a.events.Collect()
	}

	d := a.drivers.Get(req.UMF)
	compls, err := d.Update(ctx, opts, global, req.SelectedBuffer, requestCompletions, metricFn)
	if err != nil {
		resp.Error = err.Error()
		resp.HTTPStatus = http.StatusInternalServerError
		return resp
	}
	resp.Completions = compls

	language := lang.FromFilename(req.Filename)
	func() {
		a.lastUsedLock.Lock()
		defer a.lastUsedLock.Unlock()
		a.lastUsed[language] = time.Now()
	}()

	return resp
}

// Reset clears state
func (a API) Reset() {
	a.drivers.Reset()
	lexicalproviders.Reset()
}

// PushEditorEvent stores the event
func (a API) PushEditorEvent(evt *component.EditorEvent) {
	if a.events != nil {
		a.events.Push(evt)
	}
}
