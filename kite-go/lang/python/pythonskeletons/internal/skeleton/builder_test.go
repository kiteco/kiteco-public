package skeleton

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v1"
)

func assertBuild(t *testing.T, src string) Builder {
	// replace tabs with double spaces
	src = strings.Replace(src, "\t", "  ", -1)

	// parse yaml
	var raw []RawNode
	require.NoError(t, yaml.Unmarshal([]byte(src), &raw))

	// build nodes
	builder := NewBuilder()
	require.NoError(t, builder.Build(raw))
	require.NoError(t, builder.Validate())
	return builder
}

func assertTypes(t *testing.T, expected, actual []string) {
	if len(expected) != len(actual) {
		t.Errorf("lengths of types do not match")
	}

	seen := make(map[string]bool)
	for _, ty := range actual {
		if len(strings.TrimSpace(ty)) == 0 {
			t.Errorf("got empty type")
		} else {
			seen[ty] = true
		}
	}

	for _, ty := range expected {
		if !seen[ty] {
			t.Errorf("parameter missing type %s", ty)
		}
	}
}

func assertParam(t *testing.T, expected, actual Param) {
	if expected.Name != actual.Name {
		t.Errorf("wrong parameter name: %s != %s", expected.Name, actual.Name)
	}

	if expected.Default != actual.Default {
		t.Errorf("wrong parameter default: %s != %s", expected.Default, actual.Default)
	}

	assertTypes(t, expected.Types, actual.Types)
}

func assertFunction(t *testing.T, builder Builder, expected Function) {
	t.Logf("expected:\n%s\n", pretty.Sprintf("%# v", expected))
	actual := builder.Functions[expected.Path.Hash]
	if actual == nil {
		t.Errorf("Function %s not found in tables", expected.Path)
		return
	}

	t.Logf("actual:\n%s\n", pretty.Sprintf("%# v", *actual))

	if expected.Path.Hash != actual.Path.Hash {
		t.Errorf("Function path mismatch: %s != %s", expected.Path, actual.Path)
	}

	if expected.Kwargs != nil {
		if actual.Kwargs == nil {
			t.Errorf("expected non nil kwargs")
		} else {
			assertParam(t, *expected.Kwargs, *actual.Kwargs)
		}
	}

	if expected.Varargs != nil {
		if actual.Varargs == nil {
			t.Errorf("expected non nil varargs")
		} else {
			assertParam(t, *expected.Varargs, *actual.Varargs)
		}
	}

	assertTypes(t, expected.Return, actual.Return)

	if len(expected.Params) != len(actual.Params) {
		t.Errorf("parameters mismatch: \n%s\n != \n%s\n",
			pretty.Sprintf("%# v", expected.Params),
			pretty.Sprintf("%# v", actual.Params))
		return
	}

	for i := range expected.Params {
		assertParam(t, expected.Params[i], actual.Params[i])
	}
}

func TestBuilder_ModuleAttrs(t *testing.T) {
	src := `
- module:
		path: foo
		attrs:
			bar: int 
			car: str 
`

	builder := assertBuild(t, src)

	// check module
	foohash := pythonimports.NewDottedPath("foo").Hash
	mod := builder.Modules[foohash]
	assert.NotNil(t, mod)

	// check path
	assert.Equal(t, foohash, mod.Path.Hash)

	// check attrs
	assert.Equal(t, "builtins.int", mod.Attrs["bar"])
	assert.Equal(t, "builtins.str", mod.Attrs["car"])
}

func TestBuilder_ModuleTypes(t *testing.T) {
	src := `
- module:
		path: foo.bar
		types:
			bar:
				methods:
					car:
						params:
							- {name: c, types: int | str, default: foo}
					star:
						params:
							- {name: cls}
				attrs:
					foo: int
					bar: str
`
	builder := assertBuild(t, src)

	// check module
	foobarhash := pythonimports.NewDottedPath("foo.bar").Hash
	mod := builder.Modules[foobarhash]
	assert.Equal(t, foobarhash, mod.Path.Hash)

	barhash := pythonimports.NewDottedPath(mod.Types["bar"]).Hash
	bar := builder.Types[barhash]
	require.NotNil(t, bar)
	assert.Equal(t, barhash, bar.Path.Hash)

	// check methods
	carstr := bar.Methods["car"]
	assert.Equal(t, "foo.bar.bar.car", carstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(carstr),
		Params: []Param{
			Param{
				Name:  "self",
				Types: []string{"foo.bar.bar"},
			},
			Param{
				Name:    "c",
				Default: "foo",
				Types:   []string{"builtins.int", "builtins.str", "foo"},
			},
		},
	})

	starstr := bar.Methods["star"]
	assert.Equal(t, "foo.bar.bar.star", starstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(starstr),
		Params: []Param{
			Param{
				Name:  "cls",
				Types: []string{"foo.bar.bar.type"},
			},
		},
	})

	// check attrs
	assert.Equal(t, "builtins.int", bar.Attrs["foo"])
	assert.Equal(t, "builtins.str", bar.Attrs["bar"])
}

func TestBuilder_ModuleFunctions(t *testing.T) {
	src := `
- module:
		path: math
		functions:
			sum:
				varargs: {name: elems, types: list<int>}
				return: int
			divide:
				params:
					- {name: numerator, types: float}
					- {name: denominator, types: float}
				return: float
- module:
		path: flag
		functions:
			set_flags:
				kwargs: {name: flags, types: dict<str - int>}
`

	builder := assertBuild(t, src)

	mathhash := pythonimports.NewDottedPath("math").Hash
	math := builder.Modules[mathhash]
	require.NotNil(t, math)
	assert.Equal(t, mathhash, math.Path.Hash)

	sumstr := math.Functions["sum"]
	assert.Equal(t, "math.sum", sumstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(sumstr),
		Varargs: &Param{
			Name:  "elems",
			Types: []string{"builtins.list"},
		},
		Return: []string{"builtins.int"},
	})

	dividestr := math.Functions["divide"]
	assert.Equal(t, "math.divide", dividestr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(dividestr),
		Params: []Param{
			Param{
				Name:  "numerator",
				Types: []string{"builtins.float"},
			},
			Param{
				Name:  "denominator",
				Types: []string{"builtins.float"},
			},
		},
		Return: []string{"builtins.float"},
	})

	flaghash := pythonimports.NewDottedPath("flag").Hash
	flag := builder.Modules[flaghash]
	require.NotNil(t, flag)
	assert.Equal(t, flaghash, flag.Path.Hash)

	setflagsstr := flag.Functions["set_flags"]
	assert.Equal(t, "flag.set_flags", setflagsstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(setflagsstr),
		Kwargs: &Param{
			Name:  "flags",
			Types: []string{"builtins.dict"},
		},
	})

}

func TestBuilder_ModuleSubModules(t *testing.T) {
	src := `
- module:
		path: math
		submodules:
			tools:
				functions:
					sum:
						varargs: {name: elems, types: list<int>}
						return: int
`

	builder := assertBuild(t, src)

	mathhash := pythonimports.NewDottedPath("math").Hash
	math := builder.Modules[mathhash]
	require.NotNil(t, math)
	assert.Equal(t, mathhash, math.Path.Hash)
	toolshash := pythonimports.NewDottedPath("math.tools").Hash
	tools := builder.Modules[toolshash]
	require.NotNil(t, tools)
	assert.Equal(t, tools, math.SubModules["tools"])
	assert.Equal(t, toolshash, tools.Path.Hash)

	sumstr := tools.Functions["sum"]
	assert.Equal(t, "math.tools.sum", sumstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(sumstr),
		Varargs: &Param{
			Name:  "elems",
			Types: []string{"builtins.list"},
		},
		Return: []string{"builtins.int"},
	})
}

func TestBuilder_Type(t *testing.T) {
	src := `
- type:
		path: foo.bar
		bases: mar.mar | zar.zar
		methods:
			car:
				params:
					- {name: c, types: int | str, default: foo}
			star:
				params:
					- {name: cls}
			far:
				params:
					- {name: self}
				return: bool
			char: {return: self}
		attrs:
			foo: int
			bar: str
			mar: self
	`

	builder := assertBuild(t, src)

	foobarhash := pythonimports.NewDottedPath("foo.bar").Hash
	bar := builder.Types[foobarhash]
	require.NotNil(t, bar)
	assert.Equal(t, foobarhash, bar.Path.Hash)

	// check bases
	assertTypes(t, []string{"mar.mar", "zar.zar"}, bar.Bases)

	// check methods
	carstr := bar.Methods["car"]
	assert.Equal(t, "foo.bar.car", carstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(carstr),
		Params: []Param{
			Param{
				Name:  "self",
				Types: []string{"foo.bar"},
			},
			Param{
				Name:    "c",
				Default: "foo",
				Types:   []string{"builtins.int", "builtins.str", "foo"},
			},
		},
	})

	starstr := bar.Methods["star"]
	assert.Equal(t, "foo.bar.star", starstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(starstr),
		Params: []Param{
			Param{
				Name:  "cls",
				Types: []string{"foo.bar.type"},
			},
		},
	})

	farstr := bar.Methods["far"]
	assert.Equal(t, "foo.bar.far", farstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(farstr),
		Params: []Param{
			Param{
				Name:  "self",
				Types: []string{"foo.bar"},
			},
		},
		Return: []string{"builtins.bool"},
	})

	charstr := bar.Methods["char"]
	assert.Equal(t, "foo.bar.char", charstr)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath(charstr),
		Params: []Param{
			Param{
				Name:  "self",
				Types: []string{"foo.bar"},
			},
		},
		Return: []string{"foo.bar"},
	})

	// check attrs
	assert.Equal(t, "builtins.int", bar.Attrs["foo"])
	assert.Equal(t, "builtins.str", bar.Attrs["bar"])
	assert.Equal(t, "foo.bar", bar.Attrs["mar"])
}

func TestBuilder_Function(t *testing.T) {
	src := `
- function:
		path: math.sum
		params:
			- {name: x, types: float}
			- {name: yy, types: float}
		varargs: {name: args, types: list<float>}
		return: float
`

	builder := assertBuild(t, src)

	mathsumhash := pythonimports.NewDottedPath("math.sum").Hash
	sum := builder.Functions[mathsumhash]
	require.NotNil(t, sum)
	assertFunction(t, builder, Function{
		Path: pythonimports.NewDottedPath("math.sum"),
		Params: []Param{
			Param{
				Name:  "x",
				Types: []string{"builtins.float"},
			},
			Param{
				Name:  "yy",
				Types: []string{"builtins.float"},
			},
		},
		Varargs: &Param{
			Name:  "args",
			Types: []string{"builtins.list"},
		},
		Return: []string{"builtins.float"},
	})
}

func TestBuilder_Attr(t *testing.T) {
	src := `
- attr:
		path: math.pi
		type: float
- attr:
		path: math.c
		type: float
	`

	builder := assertBuild(t, src)

	pihash := pythonimports.NewDottedPath("math.pi").Hash
	pi := builder.Attrs[pihash]
	require.NotNil(t, pi)
	assert.Equal(t, pihash, pi.Path.Hash)
	assertTypes(t, []string{"builtins.float"}, []string{pi.Type})

	chash := pythonimports.NewDottedPath("math.c").Hash
	c := builder.Attrs[chash]
	require.NotNil(t, c)
	assert.Equal(t, chash, c.Path.Hash)
	assertTypes(t, []string{"builtins.float"}, []string{c.Type})
}

// test utility functions
func assertTypeString(t *testing.T, ts string, expected []string) {
	actual := parseTypes(ts)
	require.Len(t, actual, len(expected))

	t.Logf("error parsing types from %s", ts)

	// make sure not empty types
	for _, ty := range actual {
		if len(strings.TrimSpace(ty)) == 0 {
			t.Error("got empty string as type")
		}
	}

	t.Logf("expected: %s", strings.Join(expected, ", "))
	t.Logf("actual: %s", strings.Join(actual, ", "))

	// make sure types are unique
	seen := make(map[string]bool)
	for _, ty := range actual {
		if seen[ty] {
			t.Errorf("non unique type %s", ty)
		}
		seen[ty] = true
	}

	for _, ty := range expected {
		if !seen[ty] {
			t.Errorf("%30s missing", ty)
		}
	}
}

func TestParseTypes_Basic(t *testing.T) {
	// basic types
	assertTypeString(t, "int | str| float|foo", []string{
		"builtins.int",
		"builtins.str",
		"builtins.float",
		"foo",
	})
}

func TestParseTypes_NoTypeSuffix(t *testing.T) {
	// make sure that .type suffix is removed
	assertTypeString(t, "int.type | str.type", []string{
		"builtins.int",
		"builtins.str",
	})
}

func TestParseTypes_Replace(t *testing.T) {
	assertTypeString(t, "None", []string{"builtins.None.__class__"})
}

func TestParseTypes_Structured(t *testing.T) {
	assertTypeString(t, "list<int | str> | tuple<foo,bar> | dict<key | key2 - val1 | val2> | int | int | list<int>", []string{
		"builtins.list",
		"builtins.tuple",
		"builtins.dict",
		"builtins.int",
	})
}

func TestParseTypes_StructuredNested(t *testing.T) {
	assertTypeString(t, "list<int list<int>> | set<list<int>>", []string{"builtins.list", "builtins.set"})
}

func TestParseTypes_StructuredNested2(t *testing.T) {
	assertTypeString(t, "list<tuple<str, str>>", []string{"builtins.list"})
}
