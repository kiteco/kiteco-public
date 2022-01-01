package curation

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/curation/segment"
)

// SegmentsFromAnnotations translates a list of annotate.Segment objects into a list of
// curation.Segment objects, including removing region delimiters populating the Region
// attribute on the output segments.
func SegmentsFromAnnotations(in []annotate.Segment) []*Segment {
	var curRegion string
	var out []*Segment
	for _, a := range in {
		if delim, ok := a.(*annotate.RegionDelimiter); ok {
			curRegion = delim.Region
		} else {
			segment := SegmentFromAnnotation(a)
			if segment == nil {
				log.Printf("unable to convert %T to annotation", a)
				continue
			}
			segment.Region = curRegion
			out = append(out, segment)
		}
	}
	return out
}

// SegmentFromAnnotation translates an object of type annotate.Segment to an object
// of type curation.Segment. Since annotate.Segment is an interface, this translation
// uses a type switch, and if a segment type is not recognized then this function
// returns nil.
func SegmentFromAnnotation(s annotate.Segment) *Segment {
	switch s := s.(type) {
	case *annotate.CodeSegment:
		return &Segment{
			Type:            segment.Code,
			Code:            s.Code,
			BeginLineNumber: s.BeginLineNumber,
			EndLineNumber:   s.EndLineNumber,
		}
	case *annotate.PlaintextAnnotation:
		return &Segment{
			Type:       segment.Plaintext,
			Expression: s.Expression,
			Value:      s.Value,
		}
	case *annotate.DirTableAnnotation:
		return &Segment{
			Type:            segment.DirTable,
			DirTablePath:    s.Path,
			DirTableCaption: s.Caption,
			DirTableCols:    s.Cols,
			DirTableEntries: s.Entries,
		}
	case *annotate.DirTreeAnnotation:
		return &Segment{
			Type:           segment.DirTree,
			DirTreePath:    s.Path,
			DirTreeCaption: s.Caption,
			DirTreeEntries: s.Entries,
		}
	case *annotate.ImageAnnotation:
		return &Segment{
			Type:          segment.Image,
			ImagePath:     s.Path,
			ImageData:     s.Data,
			ImageEncoding: s.Encoding,
			ImageCaption:  s.Caption,
		}
	case *annotate.FileAnnotation:
		return &Segment{
			Type:        segment.File,
			FilePath:    s.Path,
			FileData:    s.Data,
			FileCaption: s.Caption,
		}
	}
	return nil
}

// RegionsFromSnippet creates a list of regions from a curated snippet
func RegionsFromSnippet(snippet *CuratedSnippet) []annotate.Region {
	return RegionsFromCode(snippet.Prelude, snippet.Code, snippet.Postlude)
}

// RegionsFromCode creates a list of regions from the prelude, main, and postlude code
func RegionsFromCode(prelude, main, postlude string) []annotate.Region {
	return []annotate.Region{
		annotate.Region{Name: "prelude", Code: prelude},
		annotate.Region{Name: "main", Code: main},
		annotate.Region{Name: "postlude", Code: postlude},
	}
}
