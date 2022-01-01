package pythoncuration

import (
	"encoding/base64"
	"log"
	"mime"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/curation/segment"
	"github.com/kiteco/kiteco/kite-go/curation/titleparser"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	dirMimeType = "application/x-directory"
)

// Result contains a curated code example along with other related
// curated examples.
type Result struct {
	Curated *curation.Example
	Related []*curation.Example
}

// --

type idCount struct {
	id    int64
	count float64
}

type byCount []*idCount

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].count < b[j].count }

// SnippetToResponse builds a response.CuratedExample response from a Snippet and Searcher.
// TODO(tarak): This is here simply to allow for passive search to take output of Searcher.Find/Search
// and turn it into a valid response. This should be collapsed with the rest of the logic during
// the active/passive search architecture refactor.
func SnippetToResponse(curated *Snippet, searcher *Searcher) *response.CuratedExample {
	var related []*curation.Example
	for _, snip := range searcher.Related(curated) {
		related = append(related, snip.Curated)
	}
	result := &Result{
		Curated: curated.Curated,
		Related: related,
	}

	resp := CurationResponseFromExample(result.Curated, "", searcher)
	resp.Type = response.CuratedExampleType
	for _, rel := range result.Related {
		relativeTitle := titleparser.RelativeTitle(rel.Snippet.Title, result.Curated.Snippet.Title)
		resp.Related = append(resp.Related, CurationPreviewResponseFromExample(rel, relativeTitle))
	}

	return resp
}

// CurationResponseFromExample constructs a response struct as expected by the UI from a curated
// code example.
func CurationResponseFromExample(curated *curation.Example, relativeTitle string, searcher *Searcher) *response.CuratedExample {
	prelude, main, postlude := ResponseFromSegments(curated.Result.Segments)
	inputFiles := ResponseFromInputFiles(curated.Result.InputFiles, searcher.sampleFiles)

	var savedAs string
	if spec, err := annotate.ParseSpec(curated.Snippet.ApparatusSpec); err == nil { // found spec
		if spec.SaveAs != "" {
			savedAs = spec.SaveAs
		}
	}
	return &response.CuratedExample{
		Type:          response.CuratedExampleType,
		ID:            curated.Snippet.SnippetID,
		SavedAs:       savedAs,
		Package:       curated.Snippet.Package,
		RelativeTitle: relativeTitle,
		Title:         text.RemoveSquareBrackets(curated.Snippet.Title),
		Prelude:       prelude,
		Main:          main,
		Postlude:      postlude,
		InputFiles:    inputFiles,
	}
}

// CurationPreviewResponseFromExample constructs a response.CuratedExamplePreview
// struct as expected by the UI from a curated code example
func CurationPreviewResponseFromExample(curated *curation.Example, relativeTitle string) *response.CuratedExamplePreview {
	return &response.CuratedExamplePreview{
		ID:            curated.Snippet.SnippetID,
		Package:       curated.Snippet.Package,
		RelativeTitle: relativeTitle,
		Title:         text.RemoveSquareBrackets(curated.Snippet.Title),
	}
}

// ResponseFromInputFiles constructs a response.InputFile struct per given input file, as
// expected by the UI.
func ResponseFromInputFiles(inputFiles []*annotate.InputFile, sampleFiles map[string][]byte) (files []*response.InputFile) {
	// Make sure not to return null list (for JSON)
	files = []*response.InputFile{}

	for _, f := range inputFiles {
		if !f.Hide {
			if f.Location == "" {
				rsegment := ResponseFromInputFile(f)
				if rsegment == nil {
					continue
				}
				files = append(files, rsegment)
			} else {
				files = append(files, ResponseFromSampleFiles(f, sampleFiles)...)
			}
		}
	}
	return
}

// ResponseFromInputFile constructs a response.InputFile struct for the given
// input file, as expected by the UI.
func ResponseFromInputFile(in *annotate.InputFile) *response.InputFile {
	out := &response.InputFile{
		Name:     in.Name,
		MimeType: mime.TypeByExtension(filepath.Ext(in.Name)),
	}
	if in.Contents != "" {
		out.ContentsBase64 = base64.StdEncoding.EncodeToString([]byte(in.Contents))
	} else if in.ContentsBase64 != "" {
		out.ContentsBase64 = in.ContentsBase64
	}
	return out
}

// ResponseFromSampleFiles searches the preloaded sample files for the given input file.
// If the input file is a directory, this will match multiple sample files and create
// one response object for each.
func ResponseFromSampleFiles(in *annotate.InputFile, sampleFiles map[string][]byte) []*response.InputFile {
	// Make sure not to return null list (for JSON)
	result := []*response.InputFile{}
	if len(in.FilesToShow) > 0 {
		for _, f := range in.FilesToShow {
			f = path.Join(in.Location, f)
			result = append(result, constructSampleFilesResponse(f, sampleFiles)...)
		}
	} else {
		result = append(result, constructSampleFilesResponse(in.Location, sampleFiles)...)
	}
	return result
}

func constructSampleFilesResponse(f string, sampleFiles map[string][]byte) []*response.InputFile {
	// Make sure not to return null list (for JSON)
	result := []*response.InputFile{}
	for name, data := range sampleFiles {
		if name == f || strings.HasPrefix(name, f+"/") {
			result = append(result, &response.InputFile{
				Name:           name,
				MimeType:       mime.TypeByExtension(filepath.Ext(name)),
				ContentsBase64: base64.StdEncoding.EncodeToString(data),
			})
		}
	}
	return result
}

// ResponseFromSegments converts an array of curation.Segment, which is the the struct stored in the
// database, to an array of response.CuratedExampleSegment, which is the struct sent to the frontend.
func ResponseFromSegments(segments []*curation.Segment) (prelude, main, postlude []*response.CuratedExampleSegment) {
	// Make sure not to return null lists (for JSON)
	prelude = []*response.CuratedExampleSegment{}
	main = []*response.CuratedExampleSegment{}
	postlude = []*response.CuratedExampleSegment{}
	for _, segment := range segments {
		rsegment := ResponseFromSegment(segment)
		if rsegment == nil {
			continue
		}
		if segment.Region == "prelude" {
			prelude = append(prelude, rsegment)
		} else if segment.Region == "main" || segment.Region == "" {
			main = append(main, rsegment)
		} else if segment.Region == "postlude" {
			postlude = append(postlude, rsegment)
		}
	}
	return
}

// ResponseFromSegment converts a curation.Segment, which is the the struct stored in the database, to
// a response.CuratedExampleSegment, which is the struct sent to the frontend.
func ResponseFromSegment(in *curation.Segment) *response.CuratedExampleSegment {
	var out response.CuratedExampleSegment

	switch in.Type {
	case segment.Code:
		out.Type = "code"
		var references []interface{}
		curation.SortReferencesByBeginEnd(in.References)
		for _, ref := range in.References {
			references = append(references, response.PythonReference{
				Begin:              ref.Begin,
				End:                ref.End,
				Original:           ref.Original,
				FullyQualifiedName: ref.FullyQualifiedName,
				Instance:           ref.Instance,
				NodeType:           ref.NodeType,
			})
		}
		out.Annotation = &response.CodeAnnotation{
			Code:       in.Code,
			References: references,
		}
	case segment.Plaintext:
		out.Type = "output"
		out.OutputType = segment.Plaintext
		out.Annotation = &response.PlaintextAnnotation{
			Expression: in.Expression,
			Value:      in.Value,
		}
	case segment.DirTable:
		out.Type = "output"
		out.OutputType = segment.DirTable

		annotation := []response.DirEntry{}
		for _, entry := range in.DirTableEntries {
			direntry := response.DirEntry{
				Name: entry.Name,
			}
			for _, col := range in.DirTableCols {
				switch col {
				case "size":
					if entry.Size == -1 {
						direntry.Size = "-"
					} else {
						direntry.Size = strconv.FormatInt(entry.Size, 10)
					}
				case "permissions":
					direntry.Permissions = entry.Permissions
				case "modified":
					direntry.Modified = formatTime(entry.Modified)
				case "created":
					direntry.Created = formatTime(entry.Created)
				case "accessed":
					direntry.Accessed = formatTime(entry.Accessed)
				case "ownerid":
					direntry.OwnerID = strconv.FormatInt(entry.OwnerID, 10)
				case "owner":
					direntry.Owner = entry.Owner
				case "groupid":
					direntry.GroupID = strconv.FormatInt(entry.GroupID, 10)
				case "group":
					direntry.Group = entry.Group
				}
			}
			annotation = append(annotation, direntry)
		}
		caption := in.DirTableCaption
		if caption == "" {
			caption = in.DirTablePath
			if caption == "." {
				caption = "Current Working Directory"
			}
		}
		out.Annotation = &response.DirTableAnnotation{
			Caption: caption,
			Entries: annotation,
		}
	case segment.DirTree:
		out.Type = "output"
		out.OutputType = segment.DirTree

		annotation := buildDirStructure(in.DirTreePath, in.DirTreeEntries)
		caption := in.DirTreeCaption
		if caption == "" {
			caption = in.DirTreePath
			if caption == "." {
				caption = "Current Working Directory"
			}
		}
		out.Annotation = &response.DirTreeAnnotation{
			Caption: caption,
			Entries: annotation,
		}
	case segment.Image:
		out.Type = "output"
		out.OutputType = segment.Image
		out.Annotation = &response.ImageAnnotation{
			Path:     in.ImagePath,
			Data:     in.ImageData,
			Encoding: in.ImageEncoding,
			Caption:  in.ImageCaption,
		}
	case segment.File:
		out.Type = "output"
		out.OutputType = segment.File
		out.Annotation = &response.FileAnnotation{
			Path:    in.FilePath,
			Data:    in.FileData,
			Caption: in.FileCaption,
		}
	default:
		log.Println("Encountered unknown segment type:", in.Type)
		return nil
	}

	return &out
}

// --

func buildDirStructure(root string, names map[string]string) *response.DirTreeListing {
	resp := newListing(root)
	for name, mimetype := range names {
		curr := resp
		path := strings.Split(stripRoot(name, root), "/")
		for _, part := range path {
			listing, ok := getOrCreateListing(part, curr.Listing)
			if !ok {
				curr.Listing = append(curr.Listing, listing)
			}
			curr = listing
		}
		curr.MimeType = mimetype
	}
	return resp
}

func stripRoot(path string, root string) string {
	return path[len(root)+1:]
}

func getOrCreateListing(name string, listings []*response.DirTreeListing) (*response.DirTreeListing, bool) {
	listing, ok := findListing(name, listings)
	if !ok {
		listing = newListing(name)
	}
	return listing, ok
}

func findListing(name string, listing []*response.DirTreeListing) (*response.DirTreeListing, bool) {
	for _, elem := range listing {
		if elem.Name == name {
			return elem, true
		}
	}
	return nil, false
}

func newListing(name string) *response.DirTreeListing {
	return &response.DirTreeListing{
		Name:     name,
		MimeType: dirMimeType,
		Listing:  []*response.DirTreeListing{},
	}
}

func formatTime(sec int64) string {
	t := time.Unix(sec, 0)
	return t.Format("Jan 02 15:04")
}
