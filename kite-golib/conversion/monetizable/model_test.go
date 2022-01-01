package monetizable

import (
	"encoding/json"
	"testing"

	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetParameters(t *testing.T) {
	readBytes, err := read()
	require.NoError(t, err)
	var readMap map[string]float64
	err = json.Unmarshal(readBytes, &readMap)
	require.NoError(t, err)

	m, err := getModel()
	require.NoError(t, err)
	paramsBytes, err := json.Marshal(m)
	require.NoError(t, err)
	var paramsMap map[string]float64
	err = json.Unmarshal(paramsBytes, &paramsMap)
	require.NoError(t, err)

	require.Equal(t, readMap, paramsMap)
}

func TestFillInputs(t *testing.T) {
	empty := Inputs{}
	filled := empty.fill()
	assert.EqualValues(t, fillUnknownAtomInstalled, *filled.AtomInstalled)
	assert.EqualValues(t, fillUnknownPyCharmInstalled, *filled.PyCharmInstalled)
	assert.EqualValues(t, fillUnknownVSCodeInstalled, *filled.VSCodeInstalled)
	assert.EqualValues(t, fillUnknownSublime3Installed, *filled.Sublime3Installed)
	assert.EqualValues(t, fillUnknownVimInstalled, *filled.VimInstalled)
	assert.EqualValues(t, fillUnknownIntelliJInstalled, *filled.IntelliJInstalled)
	assert.EqualValues(t, fillUnknownIntelliJPaid, *filled.IntelliJPaid)
	assert.EqualValues(t, fillUnknownGitFound, *filled.GitFound)
	assert.EqualValues(t, fillUnknownCPUThreads, *filled.CPUThreads)
	assert.EqualValues(t, fillUnknownOS, *filled.OS)
	assert.EqualValues(t, fillUnknownGeo, *filled.Geo)

	complete := Inputs{
		AtomInstalled:     proto.Bool(true),
		PyCharmInstalled:  proto.Bool(true),
		VSCodeInstalled:   proto.Bool(true),
		Sublime3Installed: proto.Bool(true),
		VimInstalled:      proto.Bool(false),
		IntelliJInstalled: proto.Bool(false),
		IntelliJPaid:      proto.Bool(false),
		GitFound:          proto.Bool(true),
		CPUThreads:        IntToPtr(5),
		OS:                OSToPtr(Linux),
		Geo:               GeoToPtr(USA),
	}
	filled = complete.fill()
	assert.True(t, *filled.AtomInstalled)
	assert.True(t, *filled.PyCharmInstalled)
	assert.True(t, *filled.VSCodeInstalled)
	assert.True(t, *filled.Sublime3Installed)
	assert.False(t, *filled.VimInstalled)
	assert.False(t, *filled.IntelliJInstalled)
	assert.False(t, *filled.IntelliJPaid)
	assert.True(t, *filled.GitFound)
	assert.EqualValues(t, 5, *filled.CPUThreads)
	assert.EqualValues(t, Linux, *filled.OS)
	assert.EqualValues(t, USA, *filled.Geo)
}

type logisticTC struct {
	x float64
	y float64
}

func TestLogistic(t *testing.T) {
	tcs := []logisticTC{
		logisticTC{
			x: 0,
			y: 0.5,
		},
		logisticTC{
			x: -2,
			y: 0.1192,
		},
		logisticTC{
			x: 6,
			y: 0.9975,
		},
	}

	for _, tc := range tcs {
		assert.InDelta(t, tc.y, logistic(tc.x), 1e-4)
		assert.True(t, logistic(tc.x) > 0)
		assert.InDelta(t, 1, logistic(tc.x)+logistic(-tc.x), 1e-4)
	}
}

type computeLogitsTC struct {
	inputs   filledInputs
	expected float64
}

// fillZero fills filledInputs with values that do not change logits
// to avoid panicking and for ease of testing
func (t computeLogitsTC) fillZero() filledInputs {
	if t.inputs.AtomInstalled == nil {
		t.inputs.AtomInstalled = proto.Bool(false)
	}
	if t.inputs.IntelliJInstalled == nil {
		t.inputs.IntelliJInstalled = proto.Bool(false)
	}
	if t.inputs.PyCharmInstalled == nil {
		t.inputs.PyCharmInstalled = proto.Bool(false)
	}
	if t.inputs.Sublime3Installed == nil {
		t.inputs.Sublime3Installed = proto.Bool(false)
	}
	if t.inputs.VimInstalled == nil {
		t.inputs.VimInstalled = proto.Bool(false)
	}
	if t.inputs.VSCodeInstalled == nil {
		t.inputs.VSCodeInstalled = proto.Bool(false)
	}
	if t.inputs.IntelliJPaid == nil {
		t.inputs.IntelliJPaid = proto.Bool(false)
	}
	if t.inputs.Geo == nil {
		t.inputs.Geo = GeoToPtr(nilGeo)
	}
	if t.inputs.OS == nil {
		t.inputs.OS = OSToPtr(nilOS)
	}
	if t.inputs.GitFound == nil {
		t.inputs.GitFound = proto.Bool(false)
	}
	if t.inputs.CPUThreads == nil {
		t.inputs.CPUThreads = IntToPtr(0)
	}

	return filledInputs(t.inputs)
}

func TestComputeLogits(t *testing.T) {
	tcs := []computeLogitsTC{
		computeLogitsTC{
			inputs: filledInputs{
				OS:            OSToPtr(Darwin),
				GitFound:      proto.Bool(true),
				AtomInstalled: proto.Bool(true),
			},
			expected: 10 + 9 + 6 + 4,
		},
		computeLogitsTC{
			inputs: filledInputs{
				OS:         OSToPtr(Linux),
				CPUThreads: IntToPtr(2),
				Geo:        GeoToPtr(USA),
			},
			expected: 10 + 8 + (2 * 5) + (-6),
		},
		computeLogitsTC{
			inputs: filledInputs{
				OS:                OSToPtr(Windows),
				IntelliJInstalled: proto.Bool(true),
				IntelliJPaid:      proto.Bool(true),
			},
			expected: 10 + 7 + 3 + 2,
		},
		computeLogitsTC{
			inputs: filledInputs{
				PyCharmInstalled:  proto.Bool(true),
				Sublime3Installed: proto.Bool(true),
				Geo:               GeoToPtr(China),
			},
			expected: 10 + (-2) + (-3) + (-7),
		},
		computeLogitsTC{
			inputs: filledInputs{
				VimInstalled:    proto.Bool(true),
				VSCodeInstalled: proto.Bool(true),
				Geo:             GeoToPtr(India),
			},
			expected: 10 + (-4) + (-5) + (-8),
		},
		computeLogitsTC{
			inputs: filledInputs{
				Geo: GeoToPtr(OtherGeo),
			},
			expected: 10,
		},
		computeLogitsTC{
			inputs: filledInputs{
				Geo: GeoToPtr(nilGeo),
			},
			expected: 10,
		},
		computeLogitsTC{
			inputs: filledInputs{
				OS: OSToPtr(nilOS),
			},
			expected: 10,
		},
	}

	model := linearModel{
		Intercept:         10,
		Darwin:            9,
		Linux:             8,
		Windows:           7,
		GitFound:          6,
		CPUThreads:        5,
		AtomInstalled:     4,
		IntelliJInstalled: 3,
		IntelliJPaid:      2,
		PyCharmInstalled:  -2,
		Sublime3Installed: -3,
		VimInstalled:      -4,
		VSCodeInstalled:   -5,
		USA:               -6,
		China:             -7,
		India:             -8,
	}
	for _, tc := range tcs {
		logits := model.computeLogits(tc.fillZero())
		assert.Equal(t, tc.expected, logits)
	}
}
