package response

// CodeAnnotation is an output annotation containing a chunk of source code
type CodeAnnotation struct {
	Code       string        `json:"code"`
	Wrapped    [][]string    `json:"wrapped"`
	References []interface{} `json:"references"`
}

// PlaintextAnnotation is an output annotation containing an expression and its value
type PlaintextAnnotation struct {
	Expression string `json:"expression"`
	Value      string `json:"value"`
}

// DirEntry refers to a file or folder in a directory.
type DirEntry struct {
	Name        string `json:"name,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	Size        string `json:"size,omitempty"`
	Modified    string `json:"modified,omitempty"`
	Created     string `json:"created,omitempty"`
	Accessed    string `json:"accessed,omitempty"`
	OwnerID     string `json:"ownerid,omitempty"`
	Owner       string `json:"owner,omitempty"`
	GroupID     string `json:"groupid,omitempty"`
	Group       string `json:"group,omitempty"`
}

// DirTableAnnotation is an output annotation containing information
// about files and folders in a directory.
type DirTableAnnotation struct {
	Caption string     `json:"caption"`
	Entries []DirEntry `json:"entries"`
}

// DirTreeListing is a listing of the files and folders in a directory.
type DirTreeListing struct {
	Name     string            `json:"name"`
	MimeType string            `json:"mime_type"`
	Listing  []*DirTreeListing `json:"listing"`
}

// DirTreeAnnotation is an output annotation containing information
// about the descendents of a directory.
type DirTreeAnnotation struct {
	Caption string          `json:"caption"`
	Entries *DirTreeListing `json:"entries"`
}

// ImageAnnotation is an output annotation containing an image
type ImageAnnotation struct {
	Path     string `json:"path"`
	Data     []byte `json:"data"`
	Encoding string `json:"encoding"`
	Caption  string `json:"caption"`
}

// FileAnnotation is an output annotation containing the contents of a file
type FileAnnotation struct {
	Path    string `json:"path"`
	Data    []byte `json:"data"`
	Caption string `json:"caption"`
}

// CuratedExampleSegment is a code or output segment in a curated code example
type CuratedExampleSegment struct {
	Type       string      `json:"type"`
	OutputType string      `json:"output_type"`
	Annotation interface{} `json:"content"` // Annotation is one of the XxxSegment objects above
}

// InputFile is the struct expected by the UI for an input file with data specified
// by the user via a postlude spec, or for a sample file.
type InputFile struct {
	Name            string `json:"name"`
	ContentsBase64  string `json:"contents_base64"`
	MimeType        string `json:"mime_type"`
	HighlightSyntax bool   `json:"highlight_syntax"`
}

// CuratedExample is an example that was hand written by curators.
type CuratedExample struct {
	Type          string       `json:"type"`
	ID            int64        `json:"id"`
	SavedAs       string       `json:"saved_as"`
	RelativeTitle string       `json:"relativeTitle"`
	InputFiles    []*InputFile `json:"inputFiles"`

	Collapsed bool `json:"collapsed"`

	Title    string                   `json:"title"`
	Prelude  []*CuratedExampleSegment `json:"prelude"`
	Main     []*CuratedExampleSegment `json:"code"`
	Postlude []*CuratedExampleSegment `json:"postlude"`
	Package  string                   `json:"package"`

	Related []*CuratedExamplePreview `json:"related"`
}

// CuratedExamplePreview is a compact version of an example that was hand
// written by curators
type CuratedExamplePreview struct {
	ID            int64  `json:"id"`
	RelativeTitle string `json:"relativeTitle"`
	Title         string `json:"title"`
	Package       string `json:"package"`
}
