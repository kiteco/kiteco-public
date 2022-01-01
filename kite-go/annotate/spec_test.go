package annotate

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractConsoleSpec(t *testing.T) {
	code := `
def foo():
	return 123

'''
stdin: abc
limits:
  timeout: 45
  max_output_lines: 20
  max_output_bytes: 300
'''
`
	specstr := extractPythonSpec(code)
	if !assert.NotEqual(t, "", specstr) {
		return
	}
	spec, err := ParseSpec(specstr)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "abc", spec.Stdin)
	assert.EqualValues(t, 45, spec.Limits.Timeout)
	assert.EqualValues(t, 20, spec.Limits.MaxLines)
	assert.EqualValues(t, 300, spec.Limits.MaxBytes)
	assert.Nil(t, spec.HTTPRequest)
}

func TestExtractHttpSpec(t *testing.T) {
	code := `
def foo():
	return 123

'''
http_request:
  method: "GET"
  url: "http://localhost/foo?q=bar"
  form:
    foo: bar
    ham: spam
'''
`

	specstr := extractPythonSpec(code)
	if !assert.NotEqual(t, "", specstr) {
		return
	}
	spec, err := ParseSpec(specstr)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, spec.HTTPRequest) {
		return
	}
	assert.Equal(t, "GET", spec.HTTPRequest.Method)
	assert.Equal(t, "http://localhost/foo?q=bar", spec.HTTPRequest.URL)
	assert.EqualValues(t, map[string]string{"foo": "bar", "ham": "spam"}, spec.HTTPRequest.Form)
}

func TestExtractFilesSpec(t *testing.T) {
	code := `
'''
http_request:
  method: "GET"
  url: "http://localhost/foo?q=bar"
  files:
  - field: ham
    path: spam.png
    data: sauce
  - field: wam
    path: bam.png
    data: please
'''
`

	specstr := extractPythonSpec(code)
	require.NotEqual(t, "", specstr)
	spec, err := ParseSpec(specstr)
	require.NoError(t, err)
	require.NotNil(t, spec.HTTPRequest)

	assert.Equal(t, "GET", spec.HTTPRequest.Method)
	assert.Equal(t, "http://localhost/foo?q=bar", spec.HTTPRequest.URL)

	require.Len(t, spec.HTTPRequest.Files, 2)

	assert.Equal(t, "ham", spec.HTTPRequest.Files[0].Field)
	assert.Equal(t, "spam.png", spec.HTTPRequest.Files[0].Path)
	assert.Equal(t, "sauce", string(spec.HTTPRequest.Files[0].Data))

	assert.Equal(t, "wam", spec.HTTPRequest.Files[1].Field)
	assert.Equal(t, "bam.png", spec.HTTPRequest.Files[1].Path)
	assert.Equal(t, "please", string(spec.HTTPRequest.Files[1].Data))
}

func TestExtractPythonSpec_UserDefined(t *testing.T) {
	code := `
a = '''
http_request:
  method: "GET"
  url: "http://localhost/foo?q=bar"
'''
`
	specstr := extractPythonSpec(code)
	require.Equal(t, "", specstr)
}

func TestExtractPythonSpec_NotAtEnd(t *testing.T) {
	code := `
'''
http_request:
  method: "GET"
  url: "http://localhost/foo?q=bar"
'''
this is random code
`
	specstr := extractPythonSpec(code)
	require.Equal(t, "", specstr)
}

func TestValidateSpecNegativeOutputLimit(t *testing.T) {
	spec := Spec{
		Limits: &sandbox.Limits{MaxBytes: -100},
	}
	issues := validateSpec(&spec)
	assert.Len(t, issues, 1)
}

func TestValidateSpecMissingURL(t *testing.T) {
	spec := Spec{
		HTTPRequest: &request{},
	}
	issues := validateSpec(&spec)
	assert.Len(t, issues, 1)
}

func TestBuildApparatus_Console(t *testing.T) {
	spec := Spec{
		Stdin: "abc",
	}
	_, err := spec.BuildApparatus()
	require.NoError(t, err)
}

func TestBuildApparatus_HTTP(t *testing.T) {
	spec := Spec{
		HTTPRequest: &request{
			Method: "GET",
			URL:    "http://localhost/foo",
		},
	}
	apparatus, err := spec.BuildApparatus()
	require.NoError(t, err)
	require.NotNil(t, apparatus)
	require.IsType(t, &sandbox.DoThenCancel{}, apparatus.Action())

	action := apparatus.Action().(*sandbox.DoThenCancel)
	require.IsType(t, &sandbox.HTTPAction{}, action.Action)

	request := action.Action.(*sandbox.HTTPAction).Request
	require.NotNil(t, request)
	assert.Equal(t, "GET", request.Method)
	assert.Equal(t, "http://localhost/foo", request.URL.String())
}

func TestBuildApparatus_InputFile(t *testing.T) {
	spec := Spec{
		InputFiles: []*InputFile{
			&InputFile{
				Name:     "a.txt",
				Contents: "test file A",
			},
			&InputFile{
				Name:           "b.txt",
				ContentsBase64: "dGVzdCBmaWxlIEI=",
			},
		},
	}

	apparatus, err := spec.BuildApparatus()
	require.NoError(t, err)

	assert.Equal(t, "test file A", string(apparatus.File("a.txt")))
	assert.Equal(t, "test file B", string(apparatus.File("b.txt")))
}

func TestBuildApparatus_MissingInputFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	SamplesDir = dir

	spec := Spec{
		InputFiles: []*InputFile{
			&InputFile{
				Name:     "",
				Location: "c.txt",
			},
		},
	}

	_, err = spec.BuildApparatus()
	require.Error(t, err)
}

func TestRemoveSpecFromPythonCode(t *testing.T) {
	assert.Equal(t, "abc\n", RemoveSpecFromCode(lang.Python, "abc\n'''foo'''"))
	assert.Equal(t, "abc\ndef'''foo'''", RemoveSpecFromCode(lang.Python, "abc\ndef'''foo'''"))
	assert.Equal(t, "abc", RemoveSpecFromCode(lang.Python, "abc"))
	assert.Equal(t, "'''abc'''xyz", RemoveSpecFromCode(lang.Python, "'''abc'''xyz"))
}

func TestNewSpecFromCode_NoSpec(t *testing.T) {
	code := `
def foo():
	return 123
`

	spec, err := NewSpecFromCode(lang.Python, code)
	require.NoError(t, err)

	assert.Equal(t, sandbox.DefaultLimits.MaxLines, spec.Limits.MaxLines,
		"expected max lines to be equal to the default")
	assert.Equal(t, sandbox.DefaultLimits.MaxBytes, spec.Limits.MaxBytes,
		"expected max bytes to be equal to the default")
}

func TestNewSpecFromCode_Spec(t *testing.T) {
	code := `
def foo():
	return 123

'''
stdin: abc
limits:
  timeout: 45
  max_output_lines: 20
  max_output_bytes: 300
'''
`

	spec, err := NewSpecFromCode(lang.Python, code)
	require.NoError(t, err)

	assert.Equal(t, 20, spec.Limits.MaxLines,
		"expected max lines to be the value in the spec")
	assert.Equal(t, 300, spec.Limits.MaxBytes,
		"expected max bytes to be the value in the spec")
}
