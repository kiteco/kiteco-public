package golang

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
	"github.com/stretchr/testify/require"
)

func TestRender_LiteralValue_Slice(t *testing.T) {
	input1 := `
func main() {
	a := []int{x,^}
}`
	comp1 := `y, z`
	expected1 := ` y, z`
	require.Equal(t, expected1, rendered(t, input1, comp1, render.MatchEnd))

	input2 := `
func main() {
	a := []int{
		x,^}
}`
	comp2 := `y, z`
	expected2 := `
		y,
		z`
	require.Equal(t, expected2, rendered(t, input2, comp2, render.MatchEnd))

	input3 := `
func main() {
	a := []int{^}
}`
	comp3 := `x, y, z`
	expected3 := `x, y, z`
	require.Equal(t, expected3, rendered(t, input3, comp3, render.MatchEnd))
}

func TestPrettify_LiteralValue_Map(t *testing.T) {
	input1 := `
type something struct {
	name string
	id int
}
func main() {
	a := map[string]bool{"cat": true, ^}
}
func random() {}`
	comp1 := `"dog": false`
	expected1 := `"dog": false`
	require.Equal(t, expected1, rendered(t, input1, comp1, render.MatchStart))

	input2 := `
func main() {
	a := map[string]bool{
		"cat": true,^}
}`
	comp2 := `
		"dog": false`
	expected2 := `
		"dog": false`
	require.Equal(t, expected2, rendered(t, input2, comp2, render.MatchEnd))

	input3 := `
func main() {
	a := map[string]bool{^}
}`
	comp3 := `"cat": true, "dog": false`
	expected3 := `
		"cat": true,
		"dog": false`
	require.Equal(t, expected3, rendered(t, input3, comp3, render.MatchEnd))
}

func TestRender_Arguments(t *testing.T) {
	input1 := `
func main() {
	foo := bar(butterfly, bee,^)
}`
	comp1 := `ladybug, locust`
	expected1 := ` ladybug, locust`
	require.Equal(t, expected1, rendered(t, input1, comp1, render.MatchEnd))

	input2 := `
func main() {
	foo := bar(
		butterfly,
		bee,^)
}`
	comp2 := `ladybug, locust`
	expected2 := `
		ladybug,
		locust`
	require.Equal(t, expected2, rendered(t, input2, comp2, render.MatchEnd))

	input3 := `
func main() {
	foo := bar(^)
}`
	comp3 := `butterfly, bee, ladybug, locust`
	expected3 := `butterfly, bee, ladybug, locust`
	require.Equal(t, expected3, rendered(t, input3, comp3, render.MatchEnd))

	input4 := `
func main() {
	foo := bar(butterfly, bee,
		^)
}`
	comp4 := `ladybug, locust`
	expected4 := `ladybug, locust`
	require.Equal(t, expected4, rendered(t, input4, comp4, render.MatchEnd))
}

func TestRender_MultiLine(t *testing.T) {
	input1 := `
func (m *Builder) WriteTo(w io.Writer) (int64, error) {
	var entries []mapEntry
    for key, value := range ^
}`
	comp1 := `m.data {entries = append}`
	expected1 := `m.data {
		entries = append
	}`
	require.Equal(t, expected1, rendered(t, input1, comp1, render.MatchStart))
}

func rendered(t *testing.T, input, completion string, match render.MatchOption) string {
	start := strings.Index(input, "^")
	if start < 0 {
		t.Fatalf("at least one cursor position char '^' is required: %q", input)
	}
	raw := input[:start] + input[start+1:]
	end := start
	if ix := strings.Index(raw, "^"); ix >= 0 {
		end = ix
		raw = raw[:ix] + raw[ix+1:]
	}
	comp := data.Completion{
		Snippet: data.Snippet{Text: completion},
		Replace: data.Selection{Begin: start, End: end},
	}
	got := FormatCompletion(raw, comp, DefaultPrettifyConfig, match)
	return got.Text
}
