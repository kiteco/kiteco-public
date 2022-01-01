package lexicalmodels

import (
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// Model exposes a method to check if the underlying model is loaded
type Model interface {
	ModelBase
	ResetLanguage(language lang.Language)
	IsLoaded() bool
	Base() ModelBase
}

// model wraps a model to listen in on prediction and unload calls
// It implements IsLoadedModel and model
type model struct {
	ModelBase
	m          sync.Mutex
	langStatus map[lang.Language]bool
}

// withStatus creates an IsLoadedModel with language status
func withStatus(m ModelBase) *model {
	return &model{
		ModelBase:  m,
		langStatus: make(map[lang.Language]bool),
	}
}

// Base returns the underlying model
func (m *model) Base() ModelBase {
	return m.ModelBase
}

// IsLoaded returns whether the underlying model is loaded/warmed up
func (m *model) IsLoaded() bool {
	m.m.Lock()
	defer m.m.Unlock()
	for _, active := range m.langStatus {
		if active {
			return true
		}
	}
	return false
}

// ResetLanguage resets the language and unloads the model if we have all languages inactive
func (m *model) ResetLanguage(language lang.Language) {
	m.m.Lock()
	defer m.m.Unlock()
	m.langStatus[language] = false
	for _, active := range m.langStatus {
		if active {
			return
		}
	}
	m.Unload()
}

func (m *model) activateLanguage(language lang.Language) {
	m.m.Lock()
	defer m.m.Unlock()
	m.langStatus[language] = true
}

// PredictChan forwards the underlying Model's predictions
func (m *model) PredictChan(ctx kitectx.Context, i predict.Inputs) (chan predict.Predicted, chan error) {
	pchan, echan := m.ModelBase.PredictChan(ctx, i)
	fwdPred := make(chan predict.Predicted, cap(pchan))

	return fwdPred, kitectx.Go(func() error {
		defer close(fwdPred)

		select {
		case <-ctx.AbortChan():
			ctx.Abort()
		case first, ok := <-pchan:
			if ok {
				m.activateLanguage(lang.FromFilename(i.FilePath))
				emit(ctx, first, pchan, fwdPred)
			}
		}
		return <-echan
	})
}

func emit(ctx kitectx.Context, first predict.Predicted, rest chan predict.Predicted, to chan predict.Predicted) {
	to <- first
	for {
		select {
		case <-ctx.AbortChan():
			ctx.Abort()
			return
		case v, ok := <-rest:
			if !ok {
				return
			}
			to <- v
		}
	}
}

// Unload implements Model
func (m *model) Unload() {
	m.ModelBase.Unload()
}
