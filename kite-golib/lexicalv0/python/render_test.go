package python

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
	"github.com/stretchr/testify/require"
)

func TestRender_ListItems(t *testing.T) {
	// ALL in different lines
	input := `some_random_list = [
    apple,
    banana,^
]`
	comp := "orange, pear"
	expected := `
    orange,
    pear`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	// Extend the current line
	input = `some_random_list = [apple, banana, ^]`
	expected = "orange, pear"
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	// Apply heuristics
	input = `some_random_list = [^]`
	comp = `many_apples, many_bananas, many_oranges, many_pears, many_pandas`
	expected = `
    many_apples,
    many_bananas,
    many_oranges,
    many_pears,
    many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	input = `some_random_list = [many_apples, many_bananas, ^ma^]`
	comp = `many_oranges, many_pears, many_pandas`
	expected = `many_oranges, many_pears, many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	input = `some_random_list = [
    many_apples, many_bananas, ^ma^]`
	comp = `many_oranges, many_pears, many_pandas`
	expected = `many_oranges, many_pears, many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))
}

func TestRender_DictItems(t *testing.T) {
	// ALL in different lines
	input := `some_random_dict = {
    apple: monkey,
    banana: chimpanzee,^
}`
	comp := "orange: cat, pear: squirrel"
	expected := `
    orange: cat,
    pear: squirrel`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	// Extend the current line
	input = `some_random_dict = {apple: monkey, banana: chimpanzee, ^}`
	expected = "orange: cat, pear: squirrel"
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	// Apply heuristics
	input = `some_random_dict = {^}`
	comp = `apple: monkey, banana: chimpanzee, orange: cat, pear: squirrel`
	expected = `
    apple: monkey,
    banana: chimpanzee,
    orange: cat,
    pear: squirrel`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	input = `some_random_dict = {apple: monkey, banana: chimpanzee, ^or^}`
	comp = `orange: cat, pear: squirrel`
	expected = `orange: cat, pear: squirrel`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	input = `some_random_list = {
    apple: monkey, banana: chimpanzee, ^or^}`
	comp = `orange: cat, pear: squirrel`
	expected = `orange: cat, pear: squirrel`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))
}

func TestRender_Params(t *testing.T) {
	// ALL in different lines
	input := `def some_random_func(
        apple,
        banana,^
)`
	comp := "orange: Int, pear=0"
	expected := `
        orange: Int,
        pear=0`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	// Extend the current line
	input = `def some_random_func(apple, banana, ^)`
	expected = "orange: Int, pear=0"
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	// Apply heuristics
	input = `def some_random_func(^)`
	comp = `many_apples, many_bananas, many_oranges, many_pears: Int, many_pandas=0`
	expected = `
        many_apples,
        many_bananas,
        many_oranges,
        many_pears: Int,
        many_pandas=0`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	input = `def some_random_func(many_apples, many_bananas, ^ma^)`
	comp = `many_oranges, many_pears, many_pandas`
	expected = `many_oranges, many_pears, many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	input = `def some_random_func(
    many_apples, many_bananas, ^ma^)`
	comp = `many_oranges, many_pears, many_pandas`
	expected = `many_oranges, many_pears, many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))
}

func TestRender_Args(t *testing.T) {
	// ALL in different lines
	input := `some_random_func(
    apple,
    banana,^
)`
	comp := "orange, pear"
	expected := `
    orange,
    pear`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	// Respect the current setting
	input = `some_random_func(apple, banana, ^)`
	expected = "orange, pear"
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	input = `some_random_func(^, banana, orange, pear)`
	comp = "apple, melon"
	expected = "apple, melon"
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	// Apply heuristics
	input = `some_random_func(^)`
	comp = `many_apples, many_bananas, many_oranges, many_pears, many_pandas`
	expected = `
    many_apples,
    many_bananas,
    many_oranges,
    many_pears,
    many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchEnd))

	input = `some_random_func(many_apples, many_bananas, ^ma^)`
	comp = `many_oranges, many_pears, many_pandas`
	expected = `many_oranges, many_pears, many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	input = `some_random_func(
    many_apples, many_bananas, ^ma^)`
	comp = `many_oranges, many_pears, many_pandas`
	expected = `many_oranges, many_pears, many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))

	input = `some_random_func(
    many_apples, many_bananas, 
    ^ma^)`
	comp = `many_oranges, many_pears, many_pandas`
	expected = `many_oranges, many_pears, many_pandas`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))
}

func TestRender_Class(t *testing.T) {
	input := `
import scrapy

class BrickSetSpider(scrapy.Spider):
    name = ^`
	comp := `[string]
    def parse()`
	expected := `[string]
    def parse()`
	require.Equal(t, expected, rendered(t, input, comp, render.MatchStart))
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
