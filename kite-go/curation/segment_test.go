package curation

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/curation/segment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeSegment(t *testing.T) {
	s := SegmentFromAnnotation(&annotate.CodeSegment{
		Code: "print 123",
	})

	assert.Equal(t, segment.Code, s.Type)
	assert.Equal(t, "print 123", s.Code)
}

func TestPlaintextAnnotation(t *testing.T) {
	s := SegmentFromAnnotation(&annotate.PlaintextAnnotation{
		Expression: "a",
		Value:      "xyz",
	})

	assert.Equal(t, segment.Plaintext, s.Type)
	assert.Equal(t, "a", s.Expression)
	assert.Equal(t, "xyz", s.Value)
}

func TestImageAnnotation(t *testing.T) {
	s := SegmentFromAnnotation(&annotate.ImageAnnotation{
		Path:     "a.png",
		Data:     []byte("imagedata"),
		Encoding: "application/png",
		Caption:  "b",
	})

	assert.Equal(t, segment.Image, s.Type)
	assert.Equal(t, "imagedata", string(s.ImageData))
	assert.Equal(t, "application/png", s.ImageEncoding)
	assert.Equal(t, "b", s.ImageCaption)
}

func TestFileAnnotation(t *testing.T) {
	s := SegmentFromAnnotation(&annotate.FileAnnotation{
		Path:    "out.txt",
		Data:    []byte("test"),
		Caption: "xyz",
	})

	assert.Equal(t, segment.File, s.Type)
	assert.Equal(t, "test", string(s.FileData))
	assert.Equal(t, "xyz", s.FileCaption)
	assert.Equal(t, "out.txt", s.FilePath)
}

func TestSegmentsFromAnnotations(t *testing.T) {
	in := []annotate.Segment{
		&annotate.RegionDelimiter{Region: "prelude"},
		&annotate.CodeSegment{Code: "print x"},
		&annotate.PlaintextAnnotation{Value: "11"},
		&annotate.RegionDelimiter{Region: "main"},
		&annotate.ImageAnnotation{Path: "image.png"},
	}

	out := SegmentsFromAnnotations(in)
	require.Len(t, out, 3)

	require.NotNil(t, out[0])
	assert.Equal(t, out[0].Type, segment.Code)
	assert.Equal(t, out[0].Region, "prelude")

	require.NotNil(t, out[1])
	assert.Equal(t, out[1].Type, segment.Plaintext)
	assert.Equal(t, out[1].Region, "prelude")

	require.NotNil(t, out[2])
	assert.Equal(t, out[2].Type, segment.Image)
	assert.Equal(t, out[2].Region, "main")
}
