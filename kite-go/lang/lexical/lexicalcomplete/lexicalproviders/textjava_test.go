package lexicalproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func Test_TextJava_EmptyFile(t *testing.T) {
	src := `$`

	initModels(t, lexicalmodels.DefaultModelOptions)

	res := requireRes(t, Text{}, src, "./src.java")
	require.Empty(t, res.out)
}

func Test_TextJava_Basic(t *testing.T) {
	src := `i$`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.java")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("import"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextJava_Basic1(t *testing.T) {
	src := `
package com.example.android;

p$
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.java")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("public class"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextJava_AndroidOnCreate(t *testing.T) {
	src := `package com.example.android;

public class Main extends ListActivity implements OnClickListener {
	@Override
	protected void onCreate($)
}
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.java")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("Bundle savedInstanceState"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextJava_NoTrailingEquals(t *testing.T) {
	src := `package com.company;

public class Main {

    public static void main(String[] args) {
        args.$
    }
}
	`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.java")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("length"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("length = "),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextKotlin_EmptyFile(t *testing.T) {
	src := `$`

	initModels(t, lexicalmodels.DefaultModelOptions)

	res := requireRes(t, Text{}, src, "./src.kt")
	require.Empty(t, res.out)
}

func Test_TextKotlin_Basic(t *testing.T) {
	src := `
package com.example.android

c$
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.kt")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("class MainActivity : AppCompatActivity"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextKotlin_AutoCloseLT(t *testing.T) {
	src := `
package com.example.android

class MainActivity : AppCompatActivity() {

		lateinit private var mFamilyNameSet: ArrayS$

}
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.kt")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("ArraySet"),
		Replace: data.Selection{Begin: -6, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("ArraySet<String"),
		Replace: data.Selection{Begin: -6, End: 0},
	}))
}

func Test_TextKotlin_AndroidOnCreate(t *testing.T) {
	src := `
package com.example.android

class MainActivity : AppCompatActivity() {

		override fun onCreate($)
}
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.kt")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("savedInstanceState: Bundle"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextScala_EmptyFile(t *testing.T) {
	src := `$`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.scala")
	require.Empty(t, res.out)
}

func Test_TextScala_Basic(t *testing.T) {
	src := `package com.kite.patterns

o$
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.scala")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("object Patterns"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextScala_Tree(t *testing.T) {
	src := `package com.kite.patterns

object Patterns {

	abstract class Tree
	$
}
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.scala")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("case class"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextScala_Tree2(t *testing.T) {
	src := `package com.kite.patterns

object Patterns {

	abstract class Tree
	case class Branch(l$)
}
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.scala")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("left: Tree"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}
