package lexicalproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func Test_TextGolang_PartialFirstToken(t *testing.T) {
	src := "pack$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("package"),
		Replace: data.Selection{Begin: -4, End: 0},
	}))
}

func Test_TextGolang_NoPartialTokens(t *testing.T) {
	src := `
func ChainClientHooks(hooks ...*ClientHooks) *ClientHooks {
	if len(hooks) == 0 {
		return nil
	}
	if len(hooks) == 1 {
		return hooks[0]
	}

	return &$
}`
	initModels(t, lexicalmodels.DefaultModelOptions)

	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("Cli"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("ClientHooks"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextGolang_Semicolon(t *testing.T) {
	src := `
package main
import (
 "something"
 "anything"
)

func main() {
 for i := 0$`
	initModels(t, lexicalmodels.DefaultModelOptions)

	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("0; i"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextGolang_CloseParen(t *testing.T) {
	src := `
package main

func main() {
  f, err := os.Open(path)
  if err != nil {
    log.Fatal(err)
  }

  defer f.$
}`
	initModels(t, lexicalmodels.DefaultModelOptions)

	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("Close()"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextGolang_DotNotShown(t *testing.T) {
	src := `
type Person {
  Name string
  Age int
}

func (p *Person) SetName(name string) {
  p.$
`
	initModels(t, lexicalmodels.DefaultModelOptions)

	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("Name = name"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet(".Name"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextGolang_MidToken_Keyword(t *testing.T) {
	src := `
package main

fu$nc main() {
`

	initModels(t, lexicalmodels.DefaultModelOptions)

	res := requireRes(t, Text{}, src, "./src.go")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("func"),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("func main"),
		Replace: data.Selection{Begin: -2, End: 0},
	}))

	require.Equal(t, "fu", res.in.PredictInputs.Prefix)
}

func Test_TextGolang_MidToken_KeywordAsIdent(t *testing.T) {
	src := `package main

type typeWriter string

func (t type$)

`
	initModels(t, lexicalmodels.DefaultModelOptions)

	res := requireRes(t, Text{}, src, "./src.go")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("typeWriter"),
		Replace: data.Selection{Begin: -4, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(" typeWriter)"),
		Replace: data.Selection{Begin: -4, End: 0},
	}))

	require.Equal(t, "type", res.in.PredictInputs.Prefix)
}

func Test_TextGolang_AfterSpace(t *testing.T) {
	src := `package permissions

func (m *Manager) HandleLanguages(w $)
	`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("http"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextGolang_AfterString(t *testing.T) {
	src := `package permissions

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
)

func (m *Manager) HandleLanguages(w http.ResponseWriter, r *http.Request) {
	var resp []stri$
}
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("string"),
		Replace: data.Selection{Begin: -4, End: 0},
	}))

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("string\n _,"),
		Replace: data.Selection{Begin: -4, End: 0},
	}))
}

func Test_TextGolang_InvalidCompletionsAfterSpace(t *testing.T) {
	src1 := `
	package main

	import "pipeline"

	func main() {
		pipeline := pipeline $
	}
	`
	src2 := `
	package main

	import "pipeline"

	func main() {
		pipeline := pipeline$
	}
	`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res1, err := runProvider(t, Text{}, src1, "./src.go")
	require.NoError(t, err)

	require.False(t, res1.containsRoot(data.Completion{
		Snippet: data.NewSnippet(".New"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))

	res2, err := runProvider(t, Text{}, src2, "./src.go")
	require.NoError(t, err)

	require.True(t, res2.containsRoot(data.Completion{
		Snippet: data.NewSnippet("pipeline.New"),
		Replace: data.Selection{Begin: -8, End: 0},
	}))
}

func Test_TextGolang_CursorInsideBrackets(t *testing.T) {
	src := "var m = map[$]bool{}"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("string"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))

}

func Test_NoNewline(t *testing.T) {
	src := "package main$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.go")
	require.NoError(t, err)

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("main\r\n\r\n"),
		Replace: data.Selection{Begin: -4, End: 0},
	}))
}
