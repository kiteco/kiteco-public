package livemetrics

import (
	"fmt"
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/golang/protobuf/proto"
	"github.com/kiteco/kiteco/kite-go/diff"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
)

func TestIsMultiline(t *testing.T) {
	var tests = []struct {
		text string
		want bool
	}{
		{"", false},
		{"    ", false},
		{"     \n     ", false},
		{"if () {", false},
		{"\n    if", true},
		{"hello\n", true},
	}
	for _, test := range tests {
		rc := uint64(utf8.RuneCountInString(test.text))
		assert.Equal(t, test.want, isMultiline(test.text, rc), fmt.Sprintf("test input: %v \ntest input end", test.text))
	}
}

func TestIndelMetricString(t *testing.T) {
	var tests = []struct {
		dt   event.DiffType
		met  indelMetricType
		want string
	}{
		{
			event.DiffType_INSERT,
			total,
			"total_inserts",
		},
		{
			event.DiffType_INSERT,
			whitespace,
			"whitespace_inserts",
		},
		{
			event.DiffType_INSERT,
			multiline,
			"multiline_inserts",
		},
		{
			event.DiffType_INSERT,
			multilineWhitespace,
			"multiline_whitespace_inserts",
		},
		{
			event.DiffType_DELETE,
			total,
			"total_deletes",
		},
		{
			event.DiffType_NONE,
			occurences,
			"occurences_difftype_none",
		},
	}

	for _, test := range tests {
		assert.EqualValues(t, test.want, indelMetric{test.met, test.dt}.String())
	}
}

func TestRead(t *testing.T) {
	d := diff.NewDiffer()
	im := newIndelMetrics()
	diffs := convertDiffToPtrArray(d.Diff("func f", "func foo(){"))
	im.update(diffs)

	store, nd := im.read(false)
	stAdr := reflect.ValueOf(store).Pointer()
	imstAdr := reflect.ValueOf(im.store).Pointer()
	assert.NotEqual(t, stAdr, imstAdr, "Read failed to return a new copy (address) of the store store.")
	assert.EqualValues(t, store, im.store, "Read failed to copy values of the store")
	assert.EqualValues(t, 0, nd)

	store, nd = im.read(true)
	assert.Empty(t, im.store, "Passing clear = true to read failed to clear the store.")
	assert.NotEqual(t, stAdr, imstAdr, "Passing clear = true to read failed to return a copy of the store.")
	assert.EqualValues(t, reflect.ValueOf(store).Pointer(), imstAdr, "Passing clear = ture to read fails to return original metrics store.")
}

func TestReadAndFlatten(t *testing.T) {
	out := make(map[string]interface{})
	iml := newIndelMetricsByLang()
	d := diff.NewDiffer()
	diffs := convertDiffToPtrArray(d.Diff("let greeting = \"こんにちは!\";", "let greeting ="))
	iml.get(lang.JavaScript).update(diffs)
	iml.readAndFlatten(false, out)

	assert.EqualValues(t, 10, out["indel_javascript_total_deletes"], "ReadAndFlatten did not copy values over.")
	assert.EqualValues(t, 1, out["indel_javascript_whitespace_deletes"], "ReadAndFlatten did not copy values over.")

	out = iml.readAndFlatten(false, nil)
	assert.IsType(t, make(map[string]interface{}), out, "ReadAndFlatten passed a nil input failed to return a map[string]interface{}")
	assert.EqualValues(t, 10, out["indel_javascript_total_deletes"], "ReadAndFlatten did not copy values over.")
	assert.EqualValues(t, 1, out["indel_javascript_whitespace_deletes"], "ReadAndFlatten did not copy values over.")
}

func TestUpdate(t *testing.T) {
	var tests = []struct {
		before    string
		after     string
		lng       lang.Language
		wantStore indelStore
	}{
		{
			"let greeting = \"こんにちは!\";",
			"let greeting =",
			lang.JavaScript,
			indelStore{
				indelMetric{total, event.DiffType_DELETE}:      10,
				indelMetric{whitespace, event.DiffType_DELETE}: 1,
			},
		},
		{
			"def hello",
			"def helloWorld() {",
			lang.Python,
			indelStore{
				indelMetric{total, event.DiffType_INSERT}:      9,
				indelMetric{whitespace, event.DiffType_INSERT}: 1,
			},
		},
		{
			"func hello",
			"func hello世界() {\n\n}",
			lang.Golang,
			indelStore{
				indelMetric{total, event.DiffType_INSERT}:               9,
				indelMetric{whitespace, event.DiffType_INSERT}:          3,
				indelMetric{multiline, event.DiffType_INSERT}:           9,
				indelMetric{multilineWhitespace, event.DiffType_INSERT}: 3,
			},
		},
		{
			"func hello世界() {\n\n}",
			"func hello",
			lang.Golang,
			indelStore{
				indelMetric{total, event.DiffType_DELETE}:               9,
				indelMetric{whitespace, event.DiffType_DELETE}:          3,
				indelMetric{multiline, event.DiffType_DELETE}:           9,
				indelMetric{multilineWhitespace, event.DiffType_DELETE}: 3,
			},
		},
		{
			"func hello(){",
			"func hello(){\n    ",
			lang.Golang,
			indelStore{
				indelMetric{total, event.DiffType_INSERT}:               5,
				indelMetric{whitespace, event.DiffType_INSERT}:          5,
				indelMetric{multiline, event.DiffType_INSERT}:           0,
				indelMetric{multilineWhitespace, event.DiffType_INSERT}: 0,
			},
		},
		{
			"func windows(){",
			"func windows(){\r\n\t",
			lang.Golang,
			indelStore{
				indelMetric{total, event.DiffType_INSERT}:               3,
				indelMetric{whitespace, event.DiffType_INSERT}:          3,
				indelMetric{multiline, event.DiffType_INSERT}:           0,
				indelMetric{multilineWhitespace, event.DiffType_INSERT}: 0,
			},
		},
	}

	for _, test := range tests {
		d := diff.NewDiffer().Diff(test.before, test.after)

		diffs := convertDiffToPtrArray(d)

		m := newIndelMetricsByLang()
		lngMet := m.get(test.lng)

		if lngMet == nil {
			assert.FailNow(t, "IndelMetricsByLang failed to get")
		}

		lngMet.update(diffs)
		failing := newIndelStore()
		for met, v := range test.wantStore {
			testMetVal := lngMet.store[met]
			if v != testMetVal {
				failing[met] = testMetVal
			}
		}

		if len(failing) != 0 {
			err := fmt.Sprintf("Store does not contain expect values! \nBuffer before: %v \nBuffer after: %v \nLang: %v\n\n", test.before, test.after, test.lng.Name())
			err += fmt.Sprint("Failing metrics: \n")
			for met, v := range failing {
				err += fmt.Sprintf("%v = %v, want %v \n", met, v, test.wantStore[met])
			}
			assert.Fail(t, err)
		}
	}
}

func TestUpdateNoDiffs(t *testing.T) {
	const nTests = 3
	m := newIndelMetrics()
	var noDiff []*event.Diff
	for i := 1; i <= nTests; i++ {
		m.update(noDiff)
		assert.EqualValues(t, m.noDiffs, i, "IndelMetrics failed to record occurences of empty []*event.Diff")
	}
}

func TestUpdateNoneDiffType(t *testing.T) {
	const nTests = 3
	m := newIndelMetrics()

	for i := 1; i <= nTests; i++ {
		noneDiff := &event.Diff{
			Type: event.DiffType.Enum(event.DiffType_NONE),
			Text: proto.String("FooBarBaz"),
		}
		var noneDiffs = []*event.Diff{noneDiff}
		m.update(noneDiffs)
		assert.EqualValues(t, i, m.store[indelMetric{occurences, event.DiffType_NONE}], "IndelMetrics failed to update occurences of DiffType_NONE")
		assert.EqualValues(t, 1, len(m.store), "IndelMetrics should not update other metrics when recieving DiffType_NONE")
	}
}

// Helper function to aid testing (m.update requires []*event.Diff but Differ.Diff() returns []event.Diff)
func convertDiffToPtrArray(d []event.Diff) []*event.Diff {
	var diffs []*event.Diff
	for i := range d {
		diffs = append(diffs, &d[i])
	}
	return diffs
}
