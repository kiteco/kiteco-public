package annotate

import (
	"log"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pythonOptions = Options{
	Language:    lang.Python,
	DockerImage: "kiteco/pythonsandbox",
}

func TestAnnotateInline(t *testing.T) {
	source := `aaa
bbb
ccc
ddd
eee`

	lines := []string{
		`[[KITE[[LINE 1]]KITE]]`,
		`[[KITE[[SHOW {"type": "plaintext", "expression": "a", "value": "123"}]]KITE]]`,
		`[[KITE[[LINE 3]]KITE]]`,
		`[[KITE[[SHOW {"type": "plaintext", "expression": "a", "value": "123"}]]KITE]]`,
	}
	output := &sandbox.Result{
		Stdout:    []byte(strings.Join(lines, "\n")),
		Succeeded: true,
	}

	annotations, err := loadAnnotations(output)
	require.NoError(t, err)

	segments, err := annotateInline(source, nil, annotations)
	require.NoError(t, err)

	require.Len(t, segments, 5)

	require.IsType(t, &CodeSegment{}, segments[0])
	assert.Equal(t, "aaa\nbbb", segments[0].(*CodeSegment).Code)

	require.IsType(t, &PlaintextAnnotation{}, segments[1])

	require.IsType(t, &CodeSegment{}, segments[2])
	assert.Equal(t, "ccc\nddd", segments[2].(*CodeSegment).Code)

	require.IsType(t, &PlaintextAnnotation{}, segments[3])

	require.IsType(t, &CodeSegment{}, segments[4])
	assert.Equal(t, "eee", segments[4].(*CodeSegment).Code)
}

func TestAnnotateOutOfLine(t *testing.T) {
	source := "aaa\nbbb"

	output := &sandbox.Result{
		Stdout:    []byte("123"),
		Succeeded: true,
	}

	annotations, err := loadAnnotations(output)
	require.NoError(t, err)

	segments := annotateOutOfLine(source, output, annotations)

	for _, seg := range segments {
		t.Logf("%T: %v\n", seg, seg)
	}

	require.Len(t, segments, 2)

	require.IsType(t, &CodeSegment{}, segments[0])
	assert.Equal(t, source, segments[0].(*CodeSegment).Code)

	require.IsType(t, &PlaintextAnnotation{}, segments[1])
	assert.Equal(t, "123", segments[1].(*PlaintextAnnotation).Value)
}

func TestRunOneLine(t *testing.T) {
	if !dockerTests {
		t.Skip(`to run examples that require docker, use "go test -docker"`)
	}

	code := `print 123`
	flow, err := Run(code, pythonOptions)
	require.NoError(t, err)
	assert.Equal(t, "123\n", flow.Plain())
}

func TestRunTwoLines(t *testing.T) {
	if !dockerTests {
		t.Skip(`to run examples that require docker, use "go test -docker"`)
	}

	code := `
print 123
print 456`
	flow, err := Run(code, pythonOptions)
	require.NoError(t, err)
	assert.Len(t, flow.Segments, 5)
	assert.Equal(t, "123\n456\n", flow.Plain())

	t.Log("Segments:")
	for _, s := range flow.Segments {
		t.Logf("%T: %+v\n", s, s)
	}
	t.Log("Raw output:\n", string(flow.Raw.Stdout))
}

func TestRunTwoLinesWithExtraWhitespace(t *testing.T) {
	if !dockerTests {
		t.Skip(`to run examples that require docker, use "go test -docker"`)
	}

	code := `

print 123
print 456

`
	flow, err := Run(code, pythonOptions)
	require.NoError(t, err)
	assert.Len(t, flow.Segments, 5) // extra one because of region delimiter
	assert.Equal(t, "123\n456\n", flow.Plain())

	log.Println("Segments:")
	for _, s := range flow.Segments {
		log.Printf("%T: %+v\n", s, s)
	}
}

func TestRunWithDirectStdout(t *testing.T) {
	if !dockerTests {
		t.Skip(`to run examples that require docker, use "go test -docker"`)
	}

	code := `
import sys
print 123
sys.stdout.write("456")`
	flow, err := Run(code, pythonOptions)
	require.NoError(t, err)
	assert.Equal(t, "123\n456\n", flow.Plain())

	log.Println("Segments:")
	for _, s := range flow.Segments {
		log.Printf("%T: %+v\n", s, s)
	}
}

func TestRunWithForLoop(t *testing.T) {
	if !dockerTests {
		t.Skip(`to run examples that require docker, use "go test -docker"`)
	}

	code := `
print 123
for i in range(4, 7):
	print i
`
	flow, err := Run(code, pythonOptions)
	require.NoError(t, err)
	assert.Equal(t, "123\n4\n5\n6\n", flow.Plain())

	log.Println("Segments:")
	for _, s := range flow.Segments {
		log.Printf("%T: %+v\n", s, s)
	}
}

func TestImageAnnotationInline(t *testing.T) {
	source := "aaa\nbbb"

	stdout := `[[KITE[[SHOW {"type": "image", "line": 0, "path": "somedir/image.png", "data": "aW1hZ2Vjb250ZW50cw=="}]]KITE]]
`

	output := &sandbox.Result{
		Succeeded: true,
		Stdout:    []byte(stdout),
	}

	annotations, err := loadAnnotations(output)
	require.NoError(t, err)

	segments, err := annotateInline(source, nil, annotations)
	require.NoError(t, err)

	require.Len(t, segments, 3)

	require.IsType(t, &CodeSegment{}, segments[0])
	assert.Equal(t, "aaa", segments[0].(*CodeSegment).Code)

	require.IsType(t, &ImageAnnotation{}, segments[1])
	assert.Equal(t, "somedir/image.png", segments[1].(*ImageAnnotation).Path)
	assert.Equal(t, "imagecontents", string(segments[1].(*ImageAnnotation).Data))

	require.IsType(t, &CodeSegment{}, segments[2])
	assert.Equal(t, "bbb", segments[2].(*CodeSegment).Code)
}

func TestImageAnnotationOutOfLine(t *testing.T) {
	source := "aaa\nbbb"

	stdout := `[[KITE[[LINE 2]]KITE]]
123[[KITE[[SHOW {"type": "image", "path": "somedir/image.png", "data": "aW1hZ2Vjb250ZW50cw=="}]]KITE]]
`

	output := &sandbox.Result{
		Stdout:    []byte(stdout),
		Succeeded: true,
	}

	annotations, err := loadAnnotations(output)
	require.NoError(t, err)

	segments := annotateOutOfLine(source, output, annotations)
	for _, seg := range segments {
		t.Logf("%T: %v\n", seg, seg)
	}
	require.Len(t, segments, 3)

	require.IsType(t, &CodeSegment{}, segments[0])
	assert.Equal(t, source, segments[0].(*CodeSegment).Code)

	require.IsType(t, &PlaintextAnnotation{}, segments[1])
	assert.Equal(t, "123", segments[1].(*PlaintextAnnotation).Value)

	require.IsType(t, &ImageAnnotation{}, segments[2])
	assert.Equal(t, "somedir/image.png", segments[2].(*ImageAnnotation).Path)
	assert.Equal(t, "imagecontents", string(segments[2].(*ImageAnnotation).Data))
}

func TestRun_RemovesSpecFromSource(t *testing.T) {
	if !dockerTests {
		t.Skip(`to run examples that require docker, use "go test -docker"`)
	}

	code := `
a = 123
'''
stdin: abc
'''
`

	expected := `
a = 123

`
	flow, err := Run(code, pythonOptions)
	require.NoError(t, err)
	assert.Equal(t, expected, flow.Stencil.Presentation)
}

func TestRun_NoSpec(t *testing.T) {
	if !dockerTests {
		t.Skip(`to run examples that require docker, use "go test -docker"`)
	}

	code := "a = 123"
	flow, err := Run(code, pythonOptions)
	require.NoError(t, err)
	assert.Equal(t, code, flow.Stencil.Presentation)
}

func TestRunWithRegions(t *testing.T) {
	if !dockerTests {
		t.Skip(`to run examples that require docker, use "go test -docker"`)
	}

	regions := []Region{
		Region{"first", "print 123"},
		Region{"second", "print 456"},
	}
	flow, err := RunWithRegions(regions, "", pythonOptions)
	t.Logf("Runnable:\n%s", flow.Stencil.Runnable)
	t.Logf("Raw output:\n%s", string(flow.Raw.Stdout))
	require.NoError(t, err)
	assert.Len(t, flow.Segments, 6)
	assert.EqualValues(t, "first", flow.Segments[0].(*RegionDelimiter).Region)
	assert.EqualValues(t, "print 123", flow.Segments[1].(*CodeSegment).Code)
	assert.EqualValues(t, "123\n", flow.Segments[2].(*PlaintextAnnotation).Value)
	assert.EqualValues(t, "second", flow.Segments[3].(*RegionDelimiter).Region)
	assert.EqualValues(t, "print 456", flow.Segments[4].(*CodeSegment).Code)
	assert.EqualValues(t, "456\n", flow.Segments[5].(*PlaintextAnnotation).Value)
}
func TestFileUpload(t *testing.T) {
	if !flaskTests {
		t.Skip(`to run examples that require flask, use "go test -flask"`)
	}

	var serverCode = `
import os
from flask import Flask, request

app = Flask(__name__)

@app.route('/', methods=['POST'])
def upload_file():
    f = request.files['myfile']
    x = f.stream.read()
    return "Received field=%s, filename=%s, length=%d" % (f.name, f.filename, len(x))

app.run(port=os.environ.get("PORT", "8000"))

'''
http_request:
  url: http://localhost/
  method: POST
  files:
  - field: myfile
    path: foo.txt
    data: testdata
'''
`
	program := sandbox.NewPythonProgram(serverCode)

	httpInternalPort, _ = sandbox.UnusedPort()
	apparatus, err := NewApparatusFromCode(serverCode, lang.Python)

	require.NoError(t, err)

	result, err := apparatus.Run(program)
	if result != nil && !result.Succeeded {
		t.Log("Python said:\n" + string(result.Stderr))
	}

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Succeeded)

	require.Len(t, result.HTTPOutputs, 1)
	output := result.HTTPOutputs[0]

	require.NotNil(t, output.ResponseBody)
	assert.Equal(t, "Received field=myfile, filename=foo.txt, length=8", string(output.ResponseBody))
}

func TestCookies(t *testing.T) {
	if !flaskTests {
		t.Skip(`to run examples that require flask, use "go test -flask"`)
	}

	var serverCode = `
import os
from flask import Flask, request

app = Flask(__name__)

@app.route('/')
def upload_file():
	return request.cookies.get('ham', 'missing') + ' ' + request.cookies.get('wam', 'missing')

app.run(port=os.environ.get("PORT", "8000"))

'''
http_request:
  url: http://localhost/
  cookies:
    ham: spam
    wam: bam
'''
`
	program := sandbox.NewPythonProgram(serverCode)

	httpInternalPort, _ = sandbox.UnusedPort()
	apparatus, err := NewApparatusFromCode(serverCode, lang.Python)

	require.NoError(t, err)

	result, err := apparatus.Run(program)
	if result != nil && !result.Succeeded {
		t.Log("Python said:\n" + string(result.Stderr))
	}

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Succeeded)

	require.Len(t, result.HTTPOutputs, 1)
	output := result.HTTPOutputs[0]

	require.NotNil(t, output.ResponseBody)
	assert.Equal(t, "spam bam", string(output.ResponseBody))
}
