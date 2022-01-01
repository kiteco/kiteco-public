package lexicalproviders

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func Test_TextC_EmptyFile(t *testing.T) {
	src := `$`

	initModels(t, lexicalmodels.DefaultModelOptions)

	res := requireRes(t, Text{}, src, "./src.c")
	require.Empty(t, res.out)
}

func Test_TextC_Basic(t *testing.T) {
	src := "#include <s$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.c")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("stdio.h>"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextC_Basic1(t *testing.T) {
	src := `
	#include "stdio.h"

	struct person{
			char firstname[30];
			ch$
	};
	`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.c")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("char lastname"),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
}

func Test_TextC_Main(t *testing.T) {
	src := `
int main($)
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.c")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("int argc"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextCpp_EmptyFile(t *testing.T) {
	src := `$`

	initModels(t, lexicalmodels.DefaultModelOptions)

	res := requireRes(t, Text{}, src, "./src.cpp")
	require.Empty(t, res.out)
}

func Test_TextCpp_Basic(t *testing.T) {
	src := "#i$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.cpp")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("#include"),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
}

func Test_TextCpp_Basic1(t *testing.T) {
	src := `
#include <iostream>

i$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.cpp")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("int main"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextCpp_Main(t *testing.T) {
	src := `
int main($)
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.cpp")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("int argc"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextCSharp_EmptyFile(t *testing.T) {
	src := `$`

	initModels(t, lexicalmodels.DefaultModelOptions)

	res := requireRes(t, Text{}, src, "./src.cs")
	require.Empty(t, res.out)
}

func Test_TextCSharp_Basic(t *testing.T) {
	src := "u$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.cs")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("using System"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextCSharp_Main(t *testing.T) {
	src := `
namespace BasicsOfEntityFrameworks
{
		class Program
		{
				$
		}
}
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.cs")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("static void Main"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextCSharp_Main1(t *testing.T) {
	src := `
namespace BasicsOfEntityFrameworks
{
		class Program
		{
				static void Main($)
		}
}
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.cs")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("string[] args"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextObjC_EmptyFile(t *testing.T) {
	src := `$`

	initModels(t, lexicalmodels.DefaultModelOptions)

	res := requireRes(t, Text{}, src, "./src.m")
	require.Empty(t, res.out)
}

func Test_TextObjC_Basic(t *testing.T) {
	src := "#im$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.m")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("#import <UIKit"),
		Replace: data.Selection{Begin: -3, End: 0},
	}))
}

func Test_TextObjC_Main(t *testing.T) {
	src := `
#import <UIKit/UIKit.h>
#import "AppDelegate.h"

i$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.m")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet(fmt.Sprintf("int main(int argc%s)", data.Hole(""))),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextObjC_Basic1(t *testing.T) {
	src := `
	#import <UIKit/UIKit.h>
	#import "AppDelegate.h"
	
	int main(int argc, char *argv) {
			@$
	}
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.m")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("autoreleasepool"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextObjC_Main1(t *testing.T) {
	src := `
	#import <UIKit/UIKit.h>
	#import "AppDelegate.h"
	
	int main(int argc, char *argv) {
			@autoreleasepool {
					re$
			}
	}
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.m")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet(fmt.Sprintf("return UIApplicationMain(argc%s)", data.Hole(""))),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
}
