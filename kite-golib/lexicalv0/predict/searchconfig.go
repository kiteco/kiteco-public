package predict

import (
	"encoding/json"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// SearchConfig is the configuration for beam search
type SearchConfig struct {
	Window                int
	TopK                  int
	TopP                  float32
	MinP                  float32
	BeamWidth             int
	Depth                 int
	PrefixRegularization  float32
	UseTemperatureScaling bool
	IdentTemperature      float32
	LexicalTemperature    float32
	NumLexicalTokens      int
}

// Validate will check for invariants that must be true for a valid SearchConfig
func (s SearchConfig) Validate() error {
	if (s == SearchConfig{}) {
		return errors.Errorf("using uninitialized search config")
	}
	if s.Window <= 0 {
		return errors.Errorf("context window is set to %d, needs to be > 0", s.Window)
	}
	if s.BeamWidth <= 0 {
		return errors.Errorf("beam width is set to %d, needs to be > 0", s.BeamWidth)
	}
	if s.Depth <= 0 {
		return errors.Errorf("beam depth is set to %d, needs to be > 0", s.Depth)
	}
	if s.TopK <= 0 {
		return errors.Errorf("topK is set to %d, needs to be > 0", s.TopK)
	}
	if s.TopP <= 0 || s.TopP > 1 {
		return errors.Errorf("topP is set to %f, needs to be (0, 1]", s.TopP)
	}
	if s.MinP < 0 || s.MinP > 1 {
		return errors.Errorf("minP is set to %f, needs to be [0, 1]", s.MinP)
	}
	if s.PrefixRegularization < 0 || s.PrefixRegularization > 1 {
		return errors.Errorf("prefix regularization is set to %f, needs to be [0, 1]", s.PrefixRegularization)
	}
	if s.UseTemperatureScaling {
		if s.IdentTemperature <= 0 {
			return errors.Errorf("IdentTemperature %f must be > 0", s.IdentTemperature)
		}
		if s.LexicalTemperature <= 0 {
			return errors.Errorf("LexicalTemperature %f must be > 0", s.LexicalTemperature)
		}
	}
	return nil
}

// SearchConfigPathFromModelPath ...
func SearchConfigPathFromModelPath(path string) string {
	return fileutil.Join(path, "searchconfig.json")
}

// NewSearchConfigFromModelPath loads a search config associated with a particular model path
func NewSearchConfigFromModelPath(path string) (SearchConfig, error) {
	return NewSearchConfig(SearchConfigPathFromModelPath(path))
}

// NewSearchConfig loads a SearchConfig from a path in s3.
func NewSearchConfig(path string) (SearchConfig, error) {
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return SearchConfig{}, errors.Errorf("error reading config from '%s': %v", path, err)
	}
	defer f.Close()

	var conf SearchConfig
	if err := json.NewDecoder(f).Decode(&conf); err != nil {
		return SearchConfig{}, errors.Errorf("error decoding config from '%s': %v", path, err)
	}

	if err := conf.Validate(); err != nil {
		return SearchConfig{}, err
	}

	return conf, nil
}
