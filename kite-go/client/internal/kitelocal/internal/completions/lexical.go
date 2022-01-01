package completions

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/api"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const lexicalCompletionBlockTimeout = 200 * time.Millisecond

func (m *Manager) lexicalCompletions(ctx kitectx.Context, req data.APIRequest, metricFn data.EngineMetricsCallback) data.APIResponse {
	// Convert filename to unix-like path
	filename, err := localpath.ToUnix(req.Filename)
	if err != nil {
		return data.APIResponse{
			HTTPStatus: http.StatusInternalServerError,
			Error:      fmt.Sprintf("error converting to unix path: %s", err),
		}
	}

	// Lowercase paths on windows to ensure consistent casing.
	if runtime.GOOS == "windows" {
		filename = strings.ToLower(filename)
	}

	req.Filename = filename

	req.UserID = m.userIDs.UserID()
	req.MachineID = m.userIDs.MachineID()
	compOpts := api.NewCompleteOptions(req.APIOptions, lang.FromFilename(req.Filename))
	newRenderOpts, _ := m.cohort.Cohorts().AugmentRenderOpts(compOpts.RenderOptions)
	compOpts.RenderOptions = newRenderOpts

	// we set the timeout here so that we don't use the timeout for unit tests
	// to make sure that completions do not time out on slow machines.
	compOpts.BlockTimeout = lexicalCompletionBlockTimeout
	if m.isUnitTestMode {
		compOpts.BlockTimeout = 0
		compOpts.UnitTestMode = true
	}

	resp := m.lexicalapi.Complete(ctx, compOpts, req, nil, metricFn)
	return resp
}

// Not warming up lexical models for now https://github.com/kiteco/kiteco/issues/10070
func (m *Manager) warmupLexicalModels() {
	defer func() {
		if err := recover(); err != nil {
			rollbar.Error(errors.Errorf("panic warming up tensorflow models (lexical)"), err)
		}
	}()

	src := `
package main

import "fmt"

func main() {`

	filename := "/warmuptf.go"
	if runtime.GOOS == "windows" {
		filename = "/windows/c" + filename
		filename = strings.ToLower(filename)
	}
	cursor := len(src) - 1
	req := data.APIRequest{
		UMF: data.UMF{
			Filename:  filename,
			UserID:    m.userIDs.UserID(),
			MachineID: m.userIDs.MachineID(),
		},
		SelectedBuffer: data.SelectedBuffer{
			Buffer: data.NewBuffer(src),
			Selection: data.Selection{
				Begin: cursor,
				End:   cursor,
			},
		},
	}

	opts := api.DefaultLexicalOptions
	opts.AsyncTimeout = 10 * time.Second
	err := kitectx.Background().WithTimeout(30*time.Second, func(ctx kitectx.Context) error {
		return m.lexicalapi.Complete(ctx, opts, req, nil, nil).ToError()
	})
	if err != nil {
		log.Printf("error warming up tensorflow (lexical): %v\n", err)
	}
}

func (m *Manager) handleLexicalEvents(evt *event.Event) {
	if len(evt.Selections) != 1 {
		return
	}

	sel := evt.Selections[0]
	start := sel.GetStart()
	end := sel.GetEnd()
	if end < start {
		// Sublime has bidirectional selections
		start, end = end, start
	}

	req := data.APIRequest{
		UMF: data.UMF{
			UserID:    m.userIDs.UserID(),
			MachineID: m.userIDs.MachineID(),
			Filename:  evt.GetFilename(),
		},
		SelectedBuffer: data.NewBuffer(evt.GetText()).Select(
			data.Selection{
				Begin: int(start),
				End:   int(end),
			}),
		APIOptions: data.APIOptions{
			Editor: data.Editor(evt.GetSource()),
		},
	}
	if err := req.Validate(); err != nil {
		rollbar.Error(errors.New("invalid completions edit request for lexical"), err.Error(), req.Editor)
		return
	}

	compOpts := api.NewCompleteOptions(req.APIOptions, lang.FromFilename(req.Filename))
	newRenderOpts, _ := m.cohort.Cohorts().AugmentRenderOpts(compOpts.MixOptions.RenderOptions)
	compOpts.MixOptions.RenderOptions = newRenderOpts

	// we set the timeout here so that we don't use the timeout for unit tests
	// to make sure that completions do not time out on slow machines.
	compOpts.BlockTimeout = lexicalCompletionBlockTimeout
	if m.isUnitTestMode {
		compOpts.BlockTimeout = 0
		compOpts.UnitTestMode = true
	}

	m.lexicalapi.Edit(kitectx.Background(), compOpts, req, nil)
}
