package execution

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

type file struct {
	Field string `yaml:"field"`
	Path  string `yaml:"path"`
	Data  string `yaml:"data"`
}

// Spec is embedded in code examples as a yaml string
type Spec struct {
	HashKey string `yaml:"hash_key"` // to avoid cache hits
	SaveAs  string `yaml:"save_as"`  // file name of main source
	Timeout int    `yaml:"timeout"`  // timeout in ms
}

func (s Spec) validate() (Spec, error) {
	var errs errors.Errors
	if s.Timeout <= 0 {
		s.Timeout = int(10 * time.Second / time.Millisecond)
	}
	if s.SaveAs == "" {
		s.SaveAs = "src.py"
	}
	return s, errs
}
