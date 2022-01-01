package pythonenv

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertAddrsEqual(t *testing.T, expected, actual pythontype.Address) {
	assert.Equal(t, expected.User, actual.User)
	assert.Equal(t, expected.Machine, actual.Machine)
	assert.Equal(t, expected.File, actual.File)
	assert.Equal(t, expected.Path.String(), actual.Path.String())
}

func TestEncodeFilename(t *testing.T) {
	var filename, expected, actual string

	filename = "example.py"
	expected = "example.py"
	actual = encodeFilename(filename)
	assert.Equal(t, expected, actual)

	filename = "/path/to/example.py"
	expected = ":path:to:example.py"
	actual = encodeFilename(filename)
	assert.Equal(t, expected, actual)

	filename = "another:example.py"
	expected = "another::example.py"
	actual = encodeFilename(filename)
	assert.Equal(t, expected, actual)

	filename = "/path/to/another:example.py"
	expected = ":path:to:another::example.py"
	actual = encodeFilename(filename)
	assert.Equal(t, expected, actual)
}

func TestDecodeFilename(t *testing.T) {
	var encoded, expected, actual string

	encoded = "example.py"
	expected = "example.py"
	actual = decodeFilename(encoded)
	assert.Equal(t, expected, actual)

	encoded = ":path:to:example.py"
	expected = "/path/to/example.py"
	actual = decodeFilename(encoded)
	assert.Equal(t, expected, actual)

	encoded = "another::example.py"
	expected = "another:example.py"
	actual = decodeFilename(encoded)
	assert.Equal(t, expected, actual)

	encoded = ":path:to:another::example.py"
	expected = "/path/to/another:example.py"
	actual = decodeFilename(encoded)
	assert.Equal(t, expected, actual)
}

func TestLocator(t *testing.T) {
	var expected, actual string

	module := pythontype.NewMockValue(
		t,
		pythontype.ModuleKind,
		pythontype.Address{
			User:    123,
			Machine: "machine",
			File:    "/path/to/example.py",
		},
	)
	expected = "123;machine;:path:to:example.py;"
	actual = Locator(module)
	assert.Equal(t, expected, actual)

	typ := pythontype.NewMockValue(
		t,
		pythontype.TypeKind,
		pythontype.Address{
			User:    123,
			Machine: "machine",
			File:    "/path/to/example.py",
			Path: pythonimports.DottedPath{
				Parts: []string{"type"},
			},
		},
	)
	expected = "123;machine;:path:to:example.py;type"
	actual = Locator(typ)
	assert.Equal(t, expected, actual)

	member := pythontype.NewMockValue(
		t,
		pythontype.InstanceKind,
		pythontype.Address{
			User:    123,
			Machine: "machine",
			File:    "/path/to/example.py",
			Path: pythonimports.DottedPath{
				Parts: []string{"type", "member"},
			},
		},
	)
	expected = "123;machine;:path:to:example.py;type.member"
	actual = Locator(member)
	assert.Equal(t, expected, actual)
}

func TestSymbolLocator(t *testing.T) {
	var expected, actual string
	attr := "attr"

	module := pythontype.NewMockValue(
		t,
		pythontype.ModuleKind,
		pythontype.Address{
			User:    123,
			Machine: "machine",
			File:    "/path/to/example.py",
		},
	)
	expected = "123;machine;:path:to:example.py;;attr"
	actual = SymbolLocator(module, attr)
	assert.Equal(t, expected, actual)

	typ := pythontype.NewMockValue(
		t,
		pythontype.TypeKind,
		pythontype.Address{
			User:    123,
			Machine: "machine",
			File:    "/path/to/example.py",
			Path: pythonimports.DottedPath{
				Parts: []string{"type"},
			},
		},
	)
	expected = "123;machine;:path:to:example.py;type;attr"
	actual = SymbolLocator(typ, attr)
	assert.Equal(t, expected, actual)

	member := pythontype.NewMockValue(
		t,
		pythontype.InstanceKind,
		pythontype.Address{
			User:    123,
			Machine: "machine",
			File:    "/path/to/example.py",
			Path: pythonimports.DottedPath{
				Parts: []string{"type", "member"},
			},
		},
	)
	expected = "123;machine;:path:to:example.py;type.member;attr"
	actual = SymbolLocator(member, attr)
	assert.Equal(t, expected, actual)

	member = pythontype.NewMockValue(
		t,
		pythontype.UnknownKind,
		pythontype.Address{IsExternalRoot: true},
	)
	expected = ";;;;attr"
	actual = SymbolLocator(member, attr)
	assert.Equal(t, expected, actual)
}

func TestIsLocator(t *testing.T) {
	var input string
	var expected, actual bool

	input = "123" + locatorSep + "machine" + locatorSep + "file" + locatorSep
	expected = true
	actual = IsLocator(input)
	assert.Equal(t, expected, actual)

	input = "123" + locatorSep + "machine" + locatorSep + "file" + locatorSep + "path"
	expected = true
	actual = IsLocator(input)
	assert.Equal(t, expected, actual)

	input = "123" + locatorSep + "machine" + locatorSep + "file"
	expected = false
	actual = IsLocator(input)
	assert.Equal(t, expected, actual)
}

func TestIsSymbolLocator(t *testing.T) {
	var input string
	var expected, actual bool

	input = "123" + locatorSep + "machine" + locatorSep + "file" + locatorSep + locatorSep + "attr"
	expected = true
	actual = IsSymbolLocator(input)
	assert.Equal(t, expected, actual)

	input = "123" + locatorSep + "machine" + locatorSep + "file" + locatorSep + "path" + locatorSep + "attr"
	expected = true
	actual = IsSymbolLocator(input)
	assert.Equal(t, expected, actual)

	input = "123" + locatorSep + "machine" + locatorSep + "file"
	expected = false
	actual = IsSymbolLocator(input)
	assert.Equal(t, expected, actual)

	input = "123" + locatorSep + "machine" + locatorSep + "file" + locatorSep
	expected = false
	actual = IsSymbolLocator(input)
	assert.Equal(t, expected, actual)

	input = "123" + locatorSep + "machine" + locatorSep + "file" + locatorSep + locatorSep
	expected = false
	actual = IsSymbolLocator(input)
	assert.Equal(t, expected, actual)
}

func TestParseValueLocator(t *testing.T) {
	var input string
	var expectedAddr, actualAddr pythontype.Address
	var err error

	input = "file"
	_, err = ParseValueLocator(input)
	assert.Error(t, err)

	input = "123;machine;file;"
	expectedAddr = pythontype.Address{
		User:    123,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath(""),
	}
	actualAddr, err = ParseValueLocator(input)
	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)

	input = "123;machine;file;foo"
	expectedAddr = pythontype.Address{
		User:    123,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath("foo"),
	}
	actualAddr, err = ParseValueLocator(input)
	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)

	input = "123456789;machine;file;foo.bar"
	expectedAddr = pythontype.Address{
		User:    123456789,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath("foo.bar"),
	}
	actualAddr, err = ParseValueLocator(input)
	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
}

func TestParseSymbolLocator(t *testing.T) {
	var input, expectedAttr, actualAttr string
	var expectedAddr, actualAddr pythontype.Address
	var err error

	input = "file"
	_, _, err = ParseSymbolLocator(input)
	assert.Error(t, err)

	input = "123;machine;file;;attr"
	expectedAddr = pythontype.Address{
		User:    123,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath(""),
	}
	expectedAttr = "attr"
	actualAddr, actualAttr, err = ParseSymbolLocator(input)
	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)

	input = "123;machine;file;foo;attr"
	expectedAddr = pythontype.Address{
		User:    123,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath("foo"),
	}
	expectedAttr = "attr"
	actualAddr, actualAttr, err = ParseSymbolLocator(input)
	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)

	input = "123456789;machine;file;foo.bar;attr"
	expectedAddr = pythontype.Address{
		User:    123456789,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath("foo.bar"),
	}
	expectedAttr = "attr"
	actualAddr, actualAttr, err = ParseSymbolLocator(input)
	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)
}

func TestParseLocator(t *testing.T) {
	var input, expectedAttr, actualAttr string
	var expectedAddr, actualAddr pythontype.Address
	var err error

	input = "123;machine;file;foo.bar;"
	_, _, err = ParseLocator(input)
	assert.Error(t, err)

	input = "badint;machine;file;foo.bar"
	_, _, err = ParseLocator(input)
	assert.Error(t, err)

	input = "123;machine;file;;attr"
	expectedAddr = pythontype.Address{
		User:    123,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath(""),
	}
	expectedAttr = "attr"
	actualAddr, actualAttr, err = ParseLocator(input)

	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)

	input = "123;machine;file;foo.bar;attr"
	expectedAddr = pythontype.Address{
		User:    123,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath("foo.bar"),
	}
	expectedAttr = "attr"
	actualAddr, actualAttr, err = ParseLocator(input)

	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)

	input = "123456789;machine;file;foo.bar"
	expectedAddr = pythontype.Address{
		User:    123456789,
		Machine: "machine",
		File:    "file",
		Path:    pythonimports.NewDottedPath("foo.bar"),
	}
	expectedAttr = ""
	actualAddr, actualAttr, err = ParseLocator(input)

	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)

	input = ";;;foo.bar"
	expectedAddr = pythontype.Address{
		Path: pythonimports.NewDottedPath("foo.bar"),
	}
	expectedAttr = ""
	actualAddr, actualAttr, err = ParseLocator(input)

	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)

	input = ";;;;attr"
	expectedAddr = pythontype.Address{}
	expectedAttr = "attr"
	actualAddr, actualAttr, err = ParseLocator(input)

	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)

	input = ";;;foo.bar;attr"
	expectedAddr = pythontype.Address{
		Path: pythonimports.NewDottedPath("foo.bar"),
	}
	expectedAttr = "attr"
	actualAddr, actualAttr, err = ParseLocator(input)

	require.NoError(t, err)
	assertAddrsEqual(t, expectedAddr, actualAddr)
	assert.Equal(t, expectedAttr, actualAttr)
	actualAddr, actualAttr, err = ParseLocator(input)
}
