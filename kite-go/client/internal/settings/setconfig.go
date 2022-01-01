package settings

import (
	"encoding/json"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/remotectrl"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// RcConfigSet is the set of keys settable via remote control
var RcConfigSet = map[string]struct{}{
	AllFeaturesPro:              {},
	TrialDuration:               {},
	ConversionCohort:            {},
	RCDisabledCompletionsCTA:    {},
	RCDisabledLexicalPython:     {},
	PaywallCompletionsLimit:     {},
	PaywallCompletionsRemaining: {},
}

// Config ...
type Config struct {
	Key   string
	Value string
}

// HandleRemoteMessage implements remotectrl.Handler
func (m *Manager) HandleRemoteMessage(msg remotectrl.Message) error {
	if msg.Type == remotectrl.SetConfig {
		config, err := m.remoteSetConfig(msg)
		props := map[string]interface{}{
			config.Key: config.Value,
		}
		if err != nil {
			clienttelemetry.Event("set_config_failed", props)
		} else {
			clienttelemetry.Event("set_config_succeeded", props)
			clienttelemetry.Update(props)
		}
		return err
	}
	return nil
}

func (m *Manager) remoteSetConfig(msg remotectrl.Message) (Config, error) {
	c, err := ConfigFromJSON(msg.Payload)
	if err != nil {
		return Config{}, err
	}
	if err := m.Set(c.Key, c.Value); err != nil {
		return Config{}, err
	}
	return c, nil
}

// ConfigFromJSON ...
func ConfigFromJSON(j json.RawMessage) (Config, error) {
	var c Config
	if err := json.Unmarshal(j, &c); err != nil {
		err = errors.New("could not unmarshal remote message payload", j, err.Error())
		rollbar.Error(err)
		return Config{}, err
	}
	if _, ok := RcConfigSet[c.Key]; !ok {
		err := errors.New("key in config for set_conversion_cohort unknown")
		rollbar.Error(err)
		return Config{}, err
	}
	return c, nil
}

func (m *Manager) handleConversionCohort(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(m.cohort.ConversionCohort()))
}
