package completions

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var pseudoTimeout = 150 * time.Millisecond

func (m *Manager) state(ctx kitectx.Context, filename, text string, editor data.Editor, cursor int) (*driver.State, error) {
	ctx.CheckAbort()

	state := fmt.Sprintf("%x", md5.Sum([]byte(text)))

	var f *driver.State
	var exists bool
	f, exists = m.provider.Driver(ctx, filename, editor.String(), state)
	if !exists && len(text) > 0 {
		// Create file driver from content if it doesn't exist
		f = m.provider.DriverFromContent(ctx, filename, editor.String(), text, cursor)
	}

	if f == nil {
		// should not happen but just to be safe
		return nil, errors.Errorf("filename/editor/state not found")
	}
	return f, nil
}

func (m *Manager) idccCompletions(ctx kitectx.Context, req data.APIRequest, metricFn data.EngineMetricsCallback) data.APIResponse {
	var pyctx *python.Context
	if st, _ := m.state(ctx, req.Filename, req.Text(), req.Editor, req.Begin); st != nil {
		if ud, _ := st.FileDriver.(*python.UnifiedDriver); ud != nil {
			pyctx = ud.Context()
		}
	}

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

	opts := m.augmentOpts(api.NewCompleteOptions(req.APIOptions))
	return m.pythonapi.Complete(ctx, opts, req, pyctx, metricFn)
}

func (m *Manager) augmentOpts(opts api.CompleteOptions) api.CompleteOptions {
	newRenderOpts, lexComplDisabled := m.cohort.Cohorts().AugmentRenderOpts(opts.RenderOptions)
	opts.MixOptions.LexicalCompletionsDisabled = lexComplDisabled
	opts.RenderOptions = newRenderOpts
	opts.UnitTestMode = m.isUnitTestMode
	return opts
}

// Not warming up python models for now https://github.com/kiteco/kiteco/issues/11788
func (m *Manager) warmupPythonModels() {
	defer func() {
		if err := recover(); err != nil {
			rollbar.Error(errors.Errorf("panic warming up tensorflow models"), err)
		}
	}()

	src := `
import json

obj = {}

s = json.dumps()
	`

	toFind := "s = json.dumps()"

	filename := "/warmuptf.py"
	if runtime.GOOS == "windows" {
		filename = "/windows/c" + filename
		filename = strings.ToLower(filename)
	}
	cursor := strings.Index(src, toFind) + len(toFind) - 1
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

	opts := api.IDCCCompleteOptions
	opts.AsyncTimeout = 10 * time.Second
	err := kitectx.Background().WithTimeout(30*time.Second, func(ctx kitectx.Context) error {
		return m.pythonapi.Complete(ctx, opts, req, nil, nil).ToError()
	})
	if err != nil {
		log.Printf("error warming up tensorflow: %v\n", err)
	}
}

func (m *Manager) handlePythonEvents(evt *event.Event) {
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
		rollbar.Error(errors.New("invalid completions edit request for Python"), err.Error(), req.Editor)
		return
	}

	ctx := kitectx.Background()

	var pyctx *python.Context
	if st, _ := m.state(ctx, req.Filename, req.Text(), req.Editor, req.Begin); st != nil {
		if ud, _ := st.FileDriver.(*python.UnifiedDriver); ud != nil {
			pyctx = ud.Context()
		}
	}

	opts := m.augmentOpts(api.NewCompleteOptions(req.APIOptions))
	opts.UnitTestMode = m.isUnitTestMode
	m.pythonapi.Edit(ctx, opts, req, pyctx)
}
