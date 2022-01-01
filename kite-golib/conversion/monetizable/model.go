package monetizable

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/proto"
)

// Score ...
func Score(i Inputs) (float64, error) {
	model, err := getModel()
	if err != nil {
		return 0, err
	}
	// Missing or unknown data can be represented with nil
	// The nil values will be filled with default values
	return model.probability(i.fill()), nil
}

// Inputs ...
type Inputs filledInputs

func (i Inputs) fill() filledInputs {
	if i.AtomInstalled == nil {
		i.AtomInstalled = proto.Bool(fillUnknownAtomInstalled)
	}
	if i.IntelliJInstalled == nil {
		i.IntelliJInstalled = proto.Bool(fillUnknownIntelliJInstalled)
	}
	if i.PyCharmInstalled == nil {
		i.PyCharmInstalled = proto.Bool(fillUnknownPyCharmInstalled)
	}
	if i.Sublime3Installed == nil {
		i.Sublime3Installed = proto.Bool(fillUnknownSublime3Installed)
	}
	if i.VimInstalled == nil {
		i.VimInstalled = proto.Bool(fillUnknownVimInstalled)
	}
	if i.VSCodeInstalled == nil {
		i.VSCodeInstalled = proto.Bool(fillUnknownVSCodeInstalled)
	}
	if i.IntelliJPaid == nil {
		i.IntelliJPaid = proto.Bool(fillUnknownIntelliJPaid)
	}
	if i.Geo == nil {
		i.Geo = GeoToPtr(fillUnknownGeo)
	}
	if i.OS == nil {
		i.OS = OSToPtr(fillUnknownOS)
	}
	if i.GitFound == nil {
		i.GitFound = proto.Bool(fillUnknownGitFound)
	}
	if i.CPUThreads == nil {
		i.CPUThreads = IntToPtr(fillUnknownCPUThreads)
	}

	return filledInputs(i)
}

type filledInputs struct {
	AtomInstalled     *bool
	PyCharmInstalled  *bool
	VSCodeInstalled   *bool
	Sublime3Installed *bool
	VimInstalled      *bool
	IntelliJInstalled *bool
	IntelliJPaid      *bool
	GitFound          *bool
	CPUThreads        *int
	OS                *OS
	Geo               *Geo
}

// OS ...
type OS int

// UnmarshalText converts text to OS
func (o *OS) UnmarshalText(b []byte) error {
	str := strings.ToLower(strings.Trim(string(b), `"`))

	switch {
	case str == "darwin":
		*o = Darwin

	case str == "linux":
		*o = Linux

	case str == "windows":
		*o = Windows

	default:
		return fmt.Errorf("Invalid OS %s", string(b))
	}

	return nil
}

// OS values
const (
	nilOS OS = iota
	Darwin
	Linux
	Windows
)

// Geo ...
type Geo int

// UnmarshalText converts text to Geo
func (o *Geo) UnmarshalText(b []byte) error {
	str := strings.ToLower(strings.Trim(string(b), `"`))

	switch {
	case str == "china":
		*o = China

	case str == "india":
		*o = India

	case str == "usa":
		*o = USA

	default:
		*o = OtherGeo
	}

	return nil
}

// Geo values
const (
	nilGeo Geo = iota
	China
	India
	USA
	OtherGeo
)

const (
	fillUnknownAtomInstalled     bool = false
	fillUnknownPyCharmInstalled  bool = false
	fillUnknownVSCodeInstalled   bool = false
	fillUnknownSublime3Installed bool = false
	fillUnknownVimInstalled      bool = false
	fillUnknownIntelliJInstalled bool = false
	fillUnknownIntelliJPaid      bool = false
	fillUnknownGitFound          bool = false
	fillUnknownCPUThreads        int  = 7
	fillUnknownOS                OS   = nilOS
	fillUnknownGeo               Geo  = nilGeo
)

func (m linearModel) probability(i filledInputs) float64 {
	return logistic(m.computeLogits(i))
}

func (m linearModel) computeLogits(i filledInputs) float64 {
	logits := m.Intercept

	if *i.AtomInstalled {
		logits += m.AtomInstalled
	}
	if *i.PyCharmInstalled {
		logits += m.PyCharmInstalled
	}
	if *i.VSCodeInstalled {
		logits += m.VSCodeInstalled
	}
	if *i.Sublime3Installed {
		logits += m.Sublime3Installed
	}
	if *i.VimInstalled {
		logits += m.VimInstalled
	}
	if *i.IntelliJInstalled {
		logits += m.IntelliJInstalled
	}
	if *i.IntelliJPaid {
		logits += m.IntelliJPaid
	}
	if *i.GitFound {
		logits += m.GitFound
	}

	logits += float64(*i.CPUThreads) * m.CPUThreads

	switch *i.OS {
	case Darwin:
		logits += m.Darwin
	case Linux:
		logits += m.Linux
	case Windows:
		logits += m.Windows
	}

	switch *i.Geo {
	case China:
		logits += m.China
	case India:
		logits += m.India
	case USA:
		logits += m.USA
	}

	return logits
}

func logistic(x float64) float64 {
	return 1 / (1 + math.Exp(-x))
}

type linearModel struct {
	Intercept         float64 `json:"intercept"`
	Windows           float64 `json:"windows"`
	Darwin            float64 `json:"darwin"`
	Linux             float64 `json:"linux"`
	GitFound          float64 `json:"git_found"`
	CPUThreads        float64 `json:"cpu_threads"`
	AtomInstalled     float64 `json:"atom_installed"`
	PyCharmInstalled  float64 `json:"pycharm_installed"`
	Sublime3Installed float64 `json:"sublime3_installed"`
	VimInstalled      float64 `json:"vim_installed"`
	VSCodeInstalled   float64 `json:"vscode_installed"`
	IntelliJInstalled float64 `json:"intellij_installed"`
	IntelliJPaid      float64 `json:"intellij_paid"`
	USA               float64 `json:"USA"`
	China             float64 `json:"China"`
	India             float64 `json:"India"`
}

func read() ([]byte, error) {
	return Asset("serve/params.json")
}

var cache struct {
	model linearModel
	mu    sync.Mutex
}

func getModel() (linearModel, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.model == (linearModel{}) {
		contents, err := read()
		err = json.Unmarshal(contents, &cache.model)
		if err != nil {
			return linearModel{}, err
		}
	}
	return cache.model, nil
}

// GeoToPtr ...
func GeoToPtr(g Geo) *Geo {
	return &g
}

// OSToPtr ...
func OSToPtr(o OS) *OS {
	return &o
}

// IntToPtr ...
func IntToPtr(i int) *int {
	return &i
}
