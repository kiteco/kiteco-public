package autocorrect

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireDiff(t *testing.T, old, new string) editorapi.AutocorrectDiff {
	diffs := diffs(old, new)
	require.Len(t, diffs, 1)
	return diffs[0]
}

func assertInserted(t *testing.T, actual editorapi.AutocorrectDiff, expected ...editorapi.AutocorrectDiffLine) {
	require.Len(t, actual.Inserted, len(expected))
	for i, e := range expected {
		actual.Inserted[i].Emphasis = e.Emphasis
		assert.Equal(t, e, actual.Inserted[i], "insertions not equal:\nexpected\n%s\nactual\n%s\n", pretty.Sprintf("%#v", e), pretty.Sprintf("%#v", actual.Inserted[i]))
	}
}

func assertDeleted(t *testing.T, actual editorapi.AutocorrectDiff, expected ...editorapi.AutocorrectDiffLine) {
	require.Len(t, actual.Deleted, len(expected))
	for i, e := range expected {
		actual.Deleted[i].Emphasis = e.Emphasis
		assert.Equal(t, e, actual.Deleted[i], "deletions not equal:\nexpected\n%s\nactual\n%s\n", pretty.Sprintf("%#v", e), pretty.Sprintf("%#v", actual.Deleted[i]))
	}
}

func diffLine(line int, text string) editorapi.AutocorrectDiffLine {
	return editorapi.AutocorrectDiffLine{
		Line: line,
		Text: text,
	}
}

func TestInsert_SingleLine_Begin(t *testing.T) {
	old := "hello"
	new := "hello world"
	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(0, "hello"))
	assertInserted(t, diff, diffLine(0, "hello world"))
}

func TestDelete_SingleLine_Begin(t *testing.T) {
	old := "hello world"
	new := "world"

	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(0, "hello world"))
	assertInserted(t, diff, diffLine(0, "world"))
}

func TestInsert_SingleLine_Middle(t *testing.T) {
	old := "held"
	new := "hello world"
	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(0, "held"))
	assertInserted(t, diff, diffLine(0, "hello world"))
}

func TestDelete_SingleLine_Middle(t *testing.T) {
	old := "hello world"
	new := "held"
	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(0, "hello world"))
	assertInserted(t, diff, diffLine(0, "held"))
}

func TestInsert_SingleLine_End(t *testing.T) {
	old := "hello"
	new := "hello world"
	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(0, "hello"))
	assertInserted(t, diff, diffLine(0, "hello world"))
}

func TestInsert_MultiLine_Begin(t *testing.T) {
	old := "\nworld\n"
	new := "\nhello world\n"

	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(1, "world"))
	assertInserted(t, diff, diffLine(1, "hello world"))
}

func TestDelete_MultiLine_Begin(t *testing.T) {
	old := "\nhello world\n"
	new := "\nworld\n"

	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(1, "hello world"))
	assertInserted(t, diff, diffLine(1, "world"))
}

func TestInsert_MultiLine_Middle(t *testing.T) {
	old := "\nheld\n"
	new := "\nhello world\n"
	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(1, "held"))
	assertInserted(t, diff, diffLine(1, "hello world"))
}

func TestDelete_MultiLine_Middle(t *testing.T) {
	old := "\nhello world\n"
	new := "\nheld\n"
	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(1, "hello world"))
	assertInserted(t, diff, diffLine(1, "held"))
}
func TestInsert_MultiLine_End(t *testing.T) {
	old := "\nhello\n"
	new := "\nhello world\n"
	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(1, "hello"))
	assertInserted(t, diff, diffLine(1, "hello world"))
}

func TestDelete_MultiLine_End(t *testing.T) {
	old := "\nhello world\n"
	new := "\nhello\n"
	diff := requireDiff(t, old, new)

	assertDeleted(t, diff, diffLine(1, "hello world"))
	assertInserted(t, diff, diffLine(1, "hello"))
}

func assertEmphasis(t *testing.T, old, new string, oldEMS, newEMS []editorapi.AutocorrectLineEmphasis) {
	oldEMSActual, newEMSActual := calcEmphasis(old, new)

	t.Logf("old: '%s'\n", old)
	t.Logf("new: '%s'\n", new)

	if assert.Len(t, oldEMSActual, len(oldEMS)) {
		for i := 0; i < len(oldEMS); i++ {
			assert.Equal(t, oldEMS[i], oldEMSActual[i])
		}
	}

	if assert.Len(t, newEMSActual, len(newEMS)) {
		for i := 0; i < len(newEMS); i++ {
			assert.Equal(t, newEMS[i], newEMSActual[i])
		}
	}

}

func emphasis(start, end int) []editorapi.AutocorrectLineEmphasis {
	return []editorapi.AutocorrectLineEmphasis{
		editorapi.AutocorrectLineEmphasis{
			StartBytes: uint64(start),
			StartRunes: uint64(start),
			EndBytes:   uint64(end),
			EndRunes:   uint64(end),
		},
	}
}

func TestEmphasis_Inserted(t *testing.T) {
	old := "def oo()"
	new := "def foo()"

	newEMS := emphasis(4, 5)

	assertEmphasis(t, old, new, nil, newEMS)
}

func TestEmphasis_Deleted(t *testing.T) {
	old := "def foo())"
	new := "def foo()"

	oldEMS := emphasis(9, 10)

	assertEmphasis(t, old, new, oldEMS, nil)
}

func TestEmphasis_Substitute(t *testing.T) {
	old := "def foo(]"
	new := "def foo()"

	ems := emphasis(8, 9)

	assertEmphasis(t, old, new, ems, ems)
}
