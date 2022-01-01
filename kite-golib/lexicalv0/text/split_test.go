package text

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type splitTC struct {
	Desc     string
	Text     string
	Expected []string
}

func runSplitTCs(t *testing.T, tcs []splitTC) {
	for i, tc := range tcs {
		actual := SplitWithOpts(tc.Text, true)
		assert.Equal(t, tc.Expected, actual, "case %d: %s", i, tc.Desc)
	}
}

func Test_SplitBasic(t *testing.T) {
	tcs := []splitTC{
		{
			Desc:     "single rune",
			Text:     "r",
			Expected: []string{"r"},
		},
		{
			Desc:     "empty",
			Text:     "",
			Expected: nil,
		},
		{
			Desc:     "all letters",
			Text:     "iamletters",
			Expected: []string{"iamletters"},
		},
		{
			Desc:     "separated by spaces",
			Text:     "foo bar bax",
			Expected: []string{"foo", " ", "bar", " ", "bax"},
		},
		{
			Desc:     "keep dashes as single word",
			Text:     "foo-bar-bax",
			Expected: []string{"foo-bar-bax"},
		},
		{
			Desc:     "keep underscores as single word",
			Text:     "foo_bar_car",
			Expected: []string{"foo_bar_car"},
		},
		{
			Desc:     "numbers and text in same word",
			Text:     "foo1bar2",
			Expected: []string{"foo1bar2"},
		},
		{
			Desc:     "make sure we get the last characters after a split",
			Text:     "dog!!ball",
			Expected: []string{"dog", "!!", "ball"},
		},
		{
			Desc:     "mix underscore and dash",
			Text:     "foo-bar_baz-maz_",
			Expected: []string{"foo-bar_baz-maz_"},
		},
		{
			Desc:     "keep hspace together",
			Text:     "   \t \t",
			Expected: []string{"   \t \t"},
		},
		{
			Desc:     "keep vspace together",
			Text:     "\n\n",
			Expected: []string{"\n\n"},
		},
		{
			Desc:     "split hspace and vspace",
			Text:     "\t  \t \n",
			Expected: []string{"\t  \t ", "\n"},
		},
		{
			Desc:     "keep camel case together",
			Text:     "fooBar",
			Expected: []string{"fooBar"},
		},
		{
			Desc:     "basic numbers",
			Text:     "123 456",
			Expected: []string{"123", " ", "456"},
		},
		{
			Desc:     "numbers and letters",
			Text:     "foo1",
			Expected: []string{"foo1"},
		},
		{
			Desc:     "decimal numbers",
			Text:     "3.14",
			Expected: []string{"3", ".", "14"},
		},
		{
			Desc:     "path with slashes",
			Text:     "/foo/bar/car",
			Expected: []string{"/", "foo", "/", "bar", "/", "car"},
		},
		{
			Desc:     "paren curly brace",
			Text:     "foo(){}",
			Expected: []string{"foo", "(){}"},
		},
		{
			Desc:     "array indexing",
			Text:     "foo[bar]",
			Expected: []string{"foo", "[", "bar", "]"},
		},
		{
			Desc:     "array slicing",
			Text:     "foo[bar:car]",
			Expected: []string{"foo", "[", "bar", ":", "car", "]"},
		},
	}

	runSplitTCs(t, tcs)
}

func Test_SplitMergeKeywordsAndHSpace(t *testing.T) {
	tcs := []splitTC{
		{
			Desc:     "keyword space merged",
			Text:     ", ",
			Expected: []string{", "},
		},
		{
			Desc:     "space keyword space merged to space keyword+space",
			Text:     " # ",
			Expected: []string{" ", "# "},
		},
		{
			Desc:     "space keyword not merged",
			Text:     " !",
			Expected: []string{" ", "!"},
		},
		{
			Desc:     "for loop golang merge",
			Text:     "for i, x := range xs",
			Expected: []string{"for ", "i", ", ", "x", " ", ":= ", "range ", "xs"},
		},
		{
			Desc:     "call expression merged",
			Text:     "foo(bar, star, car)",
			Expected: []string{"foo", "(", "bar", ", ", "star", ", ", "car", ")"},
		},
		{
			Desc:     "make sure keyword and vspace don't merge",
			Text:     "for\n ",
			Expected: []string{"for", "\n", " "},
		},
	}

	runSplitTCs(t, tcs)
}

func Test_SplitJS(t *testing.T) {
	tcs := []splitTC{
		{
			Desc:     "dollar sign curly brace same word",
			Text:     "${}",
			Expected: []string{"${}"},
		},
		{
			Desc:     "hashtag ident is same word",
			Text:     "#foo",
			Expected: []string{"#foo"},
		},
		{
			Desc:     "split jsx element",
			Text:     "<button onClick={ >",
			Expected: []string{"<", "button", " ", "onClick", "={", " ", ">"},
		},
		{
			Desc:     "split jsx element",
			Text:     "<button onClick=\" >",
			Expected: []string{"<", "button", " ", "onClick", "=\"", " ", ">"},
		},
		{
			Desc:     "split jsx element",
			Text:     "<li></",
			Expected: []string{"<", "li", "></"},
		},
	}
	runSplitTCs(t, tcs)
}

func Test_Keywords(t *testing.T) {
	var kws []string
	for kw := range keywords {
		kws = append(kws, kw)
	}
	sort.Strings(kws)

	var tcs []splitTC
	for _, kw := range kws {
		tcs = append(tcs, splitTC{
			Desc:     fmt.Sprintf("ensure keyword '%s' is a single word", kw),
			Text:     kw,
			Expected: []string{kw},
		})
	}

	runSplitTCs(t, tcs)
}
