package javascript

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
)

func TestFormatCompletion(t *testing.T) {
	cases := []struct {
		// use one ^ to indicate cursor pos, an optional second ^ to indicate
		// selection end
		input      string
		inSnippet  string
		outSnippet string
		match      render.MatchOption
	}{
		{input: "a.^", inSnippet: "b", outSnippet: "b", match: render.MatchEnd},
		{input: "a(^)", inSnippet: "b", outSnippet: "b", match: render.MatchEnd},
		{input: "^a^", inSnippet: "b", outSnippet: "b", match: render.MatchEnd},
		{input: "a.^bcd^()", inSnippet: "b", outSnippet: "b", match: render.MatchEnd},
		{input: "^.a", inSnippet: "b", outSnippet: "b", match: render.MatchEnd},
		// cases from the kite-go/lang/lexical/lexicalcomplete/lexicalproviders javascript tests
		{input: "import fetch from '^'", inSnippet: "isomorphic-fetch", outSnippet: "isomorphic-fetch", match: render.MatchEnd},
		{input: `response.headers.^`, inSnippet: "append()", outSnippet: "append()", match: render.MatchEnd},
		{input: `response.headers.append(^)`, inSnippet: "response", outSnippet: "response", match: render.MatchEnd},
		{input: `response.headers.append(^)`, inSnippet: "'Content-Type'", outSnippet: "'Content-Type'", match: render.MatchEnd},
		{input: `const mapStateToProps = ^`, inSnippet: "state => ()", outSnippet: "state => ()", match: render.MatchStart},
		{input: `let alpha = 'alpha/^'`, inSnippet: "beta", outSnippet: "beta", match: render.MatchEnd},
		{input: `foo(  ^  )`, inSnippet: "bar, baz", outSnippet: "bar, baz", match: render.MatchStart},
		{input: `let alpha = 'alpha ^''`, inSnippet: "beta", outSnippet: "beta", match: render.MatchEnd},
		{input: `
<div class="footer-wrapper">
  <li>
    <a href="animateplus.com">Animate Plus<^
</div>
`, inSnippet: `/ a > < / li >`, outSnippet: "/a>\n  </li>", match: render.MatchEnd},
		{input: `
<div class="footer-wrapper">
  <li><a href="animateplus.com">Animate Plus<^
</div>
`, inSnippet: `/ a > < / li >`, outSnippet: "/a></li>", match: render.MatchEnd},
		{input: `export const LOAD_DOCS = 'load docs'
export const loadDocs = (language, identifier) => ({
  meta: {
    props:^
  }
})`, inSnippet: `{language: language}`, outSnippet: " { language: language }", match: render.MatchEnd},
		{input: `export const LOAD_DOCS = 'load docs'
export const loadDocs = (language, identifier) => ({
  meta: {
    props: ^
  }
})`, inSnippet: `{language: language}`, outSnippet: "{ language: language }", match: render.MatchStart},
		{input: `
<foo
  bar="baz"^
/>`, inSnippet: `bat="ban"`, outSnippet: `
  bat="ban"`, match: render.MatchEnd},
		{input: `
<foo
  bar="baz"
>^</foo>`, inSnippet: `Something`, outSnippet: `
  Something`, match: render.MatchEnd},
		{input: `if (language && identifier) ^`, inSnippet: `{
  return fetchDocs
}`, outSnippet: `{
  return fetchDocs
}`, match: render.MatchStart},
		// issue #10527
		{
			input:      "<div className={`form__row ^`}></div>",
			inSnippet:  "${meta.error}",
			outSnippet: "${meta.error}",
			match:      render.MatchEnd,
		},
		{
			input:      "<div className={`^`}></div>",
			inSnippet:  "form__row",
			outSnippet: "form__row",
			match:      render.MatchEnd,
		},
		{
			input:      "`string text ${^} string text`",
			inSnippet:  "x",
			outSnippet: "x",
			match:      render.MatchEnd,
		},
		{
			input:      "`inside template ${queue[i].^} etc.`",
			inSnippet:  "songTitle",
			outSnippet: "songTitle",
			match:      render.MatchEnd,
		},
		{
			input:      "`inside template ^ etc.`",
			inSnippet:  "${queue[i].songTitle}",
			outSnippet: "${queue[i].songTitle}",
			match:      render.MatchEnd,
		},
		{
			input:      "foo(a, b, c, d, e, f, ^)",
			inSnippet:  "g, h",
			outSnippet: "g, h",
			match:      render.MatchStart,
		},
		{
			input:      "foo = { a: b, c: d, e: f, ^}",
			inSnippet:  "g: h",
			outSnippet: "g: h",
			match:      render.MatchStart,
		},
		{input: `foo(^
)`, inSnippet: "arg1, arg2", outSnippet: `
  arg1,
  arg2`, match: render.MatchEnd},
		{input: `export const LOAD_DOCS = 'load docs'
export const loadDocs = (language, identifier) => ({
  meta: {
    props: {},^
  }
})`, inSnippet: `props: {}`, outSnippet: `
    props: {}`, match: render.MatchEnd},
		{input: `var = [
  ^
]`, inSnippet: "apple, banana, pear", outSnippet: `apple,
  banana,
  pear`, match: render.MatchStart},
		{input: `
^

const loadExamples = (state, action) => {
  return {}
};`, inSnippet: `const loadExamples = ()`, outSnippet: "const loadExamples = ()", match: render.MatchStart},
		{input: `const todo = (state, action) => {
  switch (action.type) {
    case Constants.UPDATE_ITEM:^
  }
}`, inSnippet: `
case Constants.TODO`, outSnippet: `
    case Constants.TODO`, match: render.MatchEnd},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			start := strings.Index(c.input, "^")
			if start < 0 {
				t.Fatalf("at least one cursor position char '^' is required: %q", c.input)
			}
			input := c.input[:start] + c.input[start+1:]
			end := start
			if ix := strings.Index(input, "^"); ix >= 0 {
				end = ix
				input = input[:ix] + input[ix+1:]
			}
			comp := data.Completion{
				Snippet: data.Snippet{Text: c.inSnippet},
				Replace: data.Selection{Begin: start, End: end},
			}

			got := FormatCompletion(input, comp, DefaultPrettifyConfig, c.match)
			if got.Text != c.outSnippet {
				t.Fatalf("want:\n%s\ngot:\n%s\n", c.outSnippet, got.Text)
			}
		})
	}
}
