package test

import (
	"context"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/response"
)

// compile-time check that we implement the intended components
var (
	_ = component.UserAuth((*panicComponent)(nil))
)

// panicInitComponent panics in Initialize()
type panicComponent struct {
	initPanic           bool
	handlersPanic       bool
	goTickPanic         bool
	terminatePanic      bool
	loggedInPanic       bool
	loggedOutPanic      bool
	pluginEventPanic    bool
	processedEventPanic bool
	eventResponsePanic  bool
	settingsPanic       bool
}

func (p *panicComponent) Name() string {
	return "panic-component"
}

func (p *panicComponent) SettingUpdated(string, string) {
	if p.settingsPanic {
		panic("setting updated failed")
	}
}

func (p *panicComponent) SettingDeleted(string) {
	if p.settingsPanic {
		panic("setting deleted failed")
	}
}

func (p *panicComponent) EventResponse(*response.Root) {
	if p.eventResponsePanic {
		panic("event response failed")
	}
}

func (p *panicComponent) ProcessedEvent(*event.Event, *component.EditorEvent) {
	if p.pluginEventPanic {
		panic("processed event failed")
	}
}

func (p *panicComponent) PluginEvent(*component.EditorEvent) {
	if p.pluginEventPanic {
		panic("plugin event failed")
	}
}

func (p *panicComponent) LoggedIn() {
	if p.loggedInPanic {
		panic("logged in failed")
	}
}

func (p *panicComponent) LoggedOut() {
	if p.loggedOutPanic {
		panic("logged out failed")
	}
}

func (p *panicComponent) Terminate() {
	if p.terminatePanic {
		panic("terminate failed")
	}
}

func (p *panicComponent) Initialize(opts component.InitializerOptions) error {
	if p.initPanic {
		panic("Initialize() failed")
	}
	return nil
}

func (p *panicComponent) RegisterHandlers(mux *mux.Router) {
	if p.handlersPanic {
		panic("RegisterHandlers failed")
	}
}

func (p *panicComponent) GoTick(ctx context.Context) {
	if p.goTickPanic {
		panic("GoTick failed")
	}
}
