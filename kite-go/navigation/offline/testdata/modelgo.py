"""
package model

import (
	"encoding/json"
	"math"
)

// Score ...
func Score(i Inputs) (float64, error) {
	model, err := getModel()
	if err != nil {
		return 0, err
	}
	return model.probability(i), nil
}

// Inputs ...
type Inputs struct {
	AtomInstalled     bool
	PyCharmInstalled  bool
	VSCodeInstalled   bool
	Sublime3Installed bool
	VimInstalled      bool
	IntelliJInstalled bool
	IntelliJPaid      bool
	GitFound          bool
	CPUThreads        int
	OS                OS
	Geo               Geo
}

// OS ...
type OS int

// OS values
const (
	nilOS OS = iota
	Darwin
	Linux
	Windows
)

// Geo ...
type Geo int

// Geo values
const (
	nilGeo Geo = iota
	China
	India
	USA
	OtherGeo
)

// Values to fill missing or unknown data.
// Values should be filled before calling Score.
const (
	FillUnknownAtomInstalled     bool = false
	FillUnknownPyCharmInstalled  bool = false
	FillUnknownVSCodeInstalled   bool = false
	FillUnknownSublime3Installed bool = false
	FillUnknownVimInstalled      bool = false
	FillUnknownIntelliJInstalled bool = false
	FillUnknownIntelliJPaid      bool = false
	FillUnknownGitFound          bool = false
	FillUnknownCPUThreads        int  = 7
	FillUnknownOS                OS   = nilOS
	FillUnknownGeo               Geo  = nilGeo
)

func (m linearModel) probability(i Inputs) float64 {
	return logistic(m.computeLogits(i))
}

func (m linearModel) computeLogits(i Inputs) float64 {
	logits := m.Intercept

	if i.AtomInstalled {
		logits += m.AtomInstalled
	}
	if i.PyCharmInstalled {
		logits += m.PyCharmInstalled
	}
	if i.VSCodeInstalled {
		logits += m.VSCodeInstalled
	}
	if i.Sublime3Installed {
		logits += m.Sublime3Installed
	}
	if i.VimInstalled {
		logits += m.VimInstalled
	}
	if i.IntelliJInstalled {
		logits += m.IntelliJInstalled
	}
	if i.IntelliJPaid {
		logits += m.IntelliJPaid
	}
	if i.GitFound {
		logits += m.GitFound
	}

	logits += float64(i.CPUThreads) * m.CPUThreads

	switch i.OS {
	case Darwin:
		logits += m.Darwin
	case Linux:
		logits += m.Linux
	case Windows:
		logits += m.Windows
	}

	switch i.Geo {
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

func getModel() (linearModel, error) {
	contents, err := read()
	var model linearModel
	err = json.Unmarshal(contents, &model)
	if err != nil {
		return linearModel{}, err
	}
	return model, nil
}
"""
