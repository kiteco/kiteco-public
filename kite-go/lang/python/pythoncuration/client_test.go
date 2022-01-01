package pythoncuration

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/curation/segment"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCuratedExampleSegments(t *testing.T) {
	segments := []*curation.Segment{
		&curation.Segment{
			Type: segment.Code,
			Code: "print 123",
		},
		&curation.Segment{
			Type: segment.Plaintext,
			Code: "abc",
		},
		&curation.Segment{
			Type:          segment.Image,
			ImagePath:     "ham/spam.jpg",
			ImageData:     []byte{1, 2, 3},
			ImageEncoding: "image/jpeg",
			ImageCaption:  "Swimming in a grilled cheese sandwich",
		},
	}

	_, r, _ := ResponseFromSegments(segments)
	require.Len(t, r, 3)
	assert.IsType(t, &response.CodeAnnotation{}, r[0].Annotation)
	assert.IsType(t, &response.PlaintextAnnotation{}, r[1].Annotation)
	assert.IsType(t, &response.ImageAnnotation{}, r[2].Annotation)
}

func TestResponseFromSegment_Code(t *testing.T) {
	r := ResponseFromSegment(&curation.Segment{
		Type: segment.Code,
		Code: "print 123",
	})
	assert.IsType(t, &response.CodeAnnotation{}, r.Annotation)
}

func TestResponseFromSegment_Plaintext(t *testing.T) {
	r := ResponseFromSegment(&curation.Segment{
		Type: segment.Plaintext,
		Code: "print 123",
	})
	assert.IsType(t, &response.PlaintextAnnotation{}, r.Annotation)
}

func TestResponseFromSegment_Image(t *testing.T) {
	r := ResponseFromSegment(&curation.Segment{
		Type:          segment.Image,
		ImagePath:     "ham/spam.jpg",
		ImageData:     []byte{1, 2, 3},
		ImageEncoding: "image/jpeg",
		ImageCaption:  "Swimming in a grilled cheese sandwich",
	})
	require.IsType(t, &response.ImageAnnotation{}, r.Annotation)
	assert.Equal(t, r.Type, "output")
	assert.Equal(t, r.OutputType, segment.Image)
	assert.Equal(t, r.Annotation.(*response.ImageAnnotation).Path, "ham/spam.jpg")
	assert.Equal(t, r.Annotation.(*response.ImageAnnotation).Encoding, "image/jpeg")
	assert.Equal(t, r.Annotation.(*response.ImageAnnotation).Data, []byte{1, 2, 3})
	assert.Equal(t, r.Annotation.(*response.ImageAnnotation).Caption, "Swimming in a grilled cheese sandwich")
}

func TestCuratedExampleSegment_File(t *testing.T) {
	r := ResponseFromSegment(&curation.Segment{
		Type:        segment.File,
		FilePath:    "output.txt",
		FileData:    []byte("ferry derry"),
		FileCaption: "Warbling with a very furry pheasant",
	})
	require.IsType(t, &response.FileAnnotation{}, r.Annotation)
	assert.Equal(t, r.Type, "output")
	assert.Equal(t, r.OutputType, segment.File)
	assert.Equal(t, r.Annotation.(*response.FileAnnotation).Path, "output.txt")
	assert.Equal(t, r.Annotation.(*response.FileAnnotation).Data, []byte("ferry derry"))
	assert.Equal(t, r.Annotation.(*response.FileAnnotation).Caption, "Warbling with a very furry pheasant")
}

func TestBuildDirStructure_AbsPath(t *testing.T) {
	names := map[string]string{
		"/kitetest/dirA":           "application/x-directory",
		"/kitetest/dirA/blahA.txt": "text/plain",
		"/kitetest/dirA/blahB.txt": "text/plain",
		"/kitetest/hello.txt":      "text/plain",
	}
	root := buildDirStructure("/kitetest", names)

	assert.Equal(t, "/kitetest", root.Name, "expected root name to be /kitetest")

	expectedChildren := map[string]string{
		"dirA":      "application/x-directory",
		"hello.txt": "text/plain",
	}
	var dirA *response.DirTreeListing
	foundChildren := make(map[string]string)
	for _, l := range root.Listing {
		foundChildren[l.Name] = l.MimeType
		if l.Name == "dirA" {
			dirA = l
		}
	}
	assert.Equal(t, expectedChildren, foundChildren, "expectedChildren and foundChildren are different")
	assert.NotNil(t, dirA, "Should have found DirTreeListing for dirA")

	expectedGrandchildren := map[string]string{
		"blahA.txt": "text/plain",
		"blahB.txt": "text/plain",
	}
	foundGrandchildren := make(map[string]string)
	for _, l := range dirA.Listing {
		foundGrandchildren[l.Name] = l.MimeType
	}
	assert.Equal(t, expectedGrandchildren, foundGrandchildren, "expectedGrandchildren and foundGrandchildren are different")
}

func TestBuildDirStructure_RelPath(t *testing.T) {
	names := map[string]string{
		"./dirA":      "application/x-directory",
		"./hello.txt": "text/plain",
	}
	root := buildDirStructure(".", names)

	assert.Equal(t, ".", root.Name, "expected root name to be .")

	expectedChildren := map[string]string{
		"dirA":      "application/x-directory",
		"hello.txt": "text/plain",
	}
	foundChildren := make(map[string]string)
	for _, l := range root.Listing {
		foundChildren[l.Name] = l.MimeType
	}
	assert.Equal(t, expectedChildren, foundChildren, "expectedChildren and foundChildren are different")
}
