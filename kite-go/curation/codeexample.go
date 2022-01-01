package curation

import (
	"database/sql"
	"html/template"
	"time"

	"github.com/kiteco/kiteco/kite-go/annotate"
	gorp "gopkg.in/gorp.v1"
)

// CuratedSnippet represents a snapshot of a curated snippet. We use an
// append-only model, where every change to a snippet adds a new row to the
// CuratedSnippet table, with an incremented SnapshotID and the same SnippetID.
type CuratedSnippet struct {
	SnippetID int64  `json:"backendId" sql:"not null" gorm:"column:SnippetID"`                  // unique snippet id
	User      string `json:"user" sql:"default:''; not null" gorm:"column:User"`                // the user who made this snapshot
	Status    string `json:"status" sql:"default:'in_progress'; not null" gorm:"column:Status"` // in_progress, pending_review, needs_attention, approved

	SnapshotID int64 `json:"snapshotId" gorm:"column:SnapshotID;primary_key"` // unique snippet id

	SnapshotTimestamp int64 `json:"-" sql:"not null" gorm:"column:SnapshotTimestamp"` // timestamp of this snapshot

	Language         string `json:"language" gorm:"column:Language"`                 // language name for this snippet
	Package          string `json:"package" gorm:"column:Package"`                   // the package that this code snippet is an exampple for
	RelevantPackages string `json:"relevant_packages"`                               // RelevantPackages contains the canonical name of packages that are relevant to Package, separated by commas
	Title            string `json:"title" sql:"type:text" gorm:"column:Title"`       // natural language title for this snippet
	Prelude          string `json:"prelude" sql:"type:text" gorm:"column:Prelude"`   // prelude of the code such as imports
	Code             string `json:"code" sql:"type:text" gorm:"column:Code"`         // the curated code example
	Postlude         string `json:"postlude" sql:"type:text" gorm:"column:Postlude"` // postlude of the code (assertions)

	Deleted    int64  `json:"deleted" gorm:"column:Deleted"` // deletion timestamp, or zero if this snippet has not been deleted
	DeletedBy  string `json:"-" gorm:"column:DeletedBy"`     // name of user that deleted this snippet
	Created    int64  `json:"-" gorm:"column:Created"`       // creation timestamp, in seconds since epoch
	CreatedBy  string `json:"-" gorm:"column:CreatedBy"`     // name of user that created this snippet
	Modified   int64  `json:"-" gorm:"column:Modified"`      // last modification timestamp, in seconds since epoch
	ModifiedBy string `json:"-" gorm:"column:ModifiedBy"`    // name of user that last modified this snippet
	Imported   int64  `json:"-" gorm:"column:Imported"`      // time of import, or zero if it was not imported from an external tool
	ImportedBy string `json:"-" gorm:"column:ImportedBy"`    // name of user that imported this snippet (if a user was authenticated)

	DisplayOrder int64 `json:"-" gorm:"column:DisplayOrder"` // the display order of this snippet in the authoring tool

	ParallelProgram string `json:"parallel_program" gorm:"column:ParallelProgram"`
	ApparatusSpec   string `json:"apparatus_spec" gorm:"column:ApparatusSpec"`

	// Ignored by the database
	ColorizedOutput template.HTML     `json:"formatted_output" sql:"-"`
	Comments        []*Comment        `json:"comments"`
	SupportingFiles []*SupportingFile `json:"supporting_files"`

	// Output represents the output of executing the curated code example. This field cannot be
	// updated via a PUT - it is populated by `Runs` when retrieved.
	Output string `json:"output" sql:"-"`
}

// TableName tells GORM to use the returned string as the name of the table backing
// CuratedSnippet instead of following GORM table name conventions.
func (c CuratedSnippet) TableName() string {
	return "CuratedSnippet"
}

// Comment is a string associated with a snippet. It is for communication between curators and is
// not visible to end users.
type Comment struct {
	ID        int64  `json:"backendId" gorm:"column:ID;primary_key"` // primary key
	SnippetID int64  `json:"snippetID" gorm:"column:SnippetID"`      // ID of associated snippet
	Text      string `json:"text" sql:"type:text" gorm:"column:Text"`

	Created   int64  `json:"createdAt" gorm:"column:Created"`   // time of creation, in seconds since epoch
	CreatedBy string `json:"createdBy" gorm:"column:CreatedBy"` // name of user that created this comment

	Modified   int64  `json:"-" gorm:"column:Modified"`   // time of last modification, in seconds since epoch
	ModifiedBy string `json:"-" gorm:"column:ModifiedBy"` // name of user that most recently modified this comment

	Dismissed   int64  `json:"dismissed" gorm:"column:Dismissed"` // time at which this comment was dismissed, or zero if not dismissed
	DismissedBy string `json:"-" gorm:"column:DismissedBy"`       // name of user that dismissed this comment
}

// TableName tells GORM to use the returned string as the name of the table backing
// Comment instead of following GORM table name conventions.
func (c Comment) TableName() string {
	return "Comment"
}

// SupportingFile is a file that is required by a snippet to run. For example, a jinja
// template. We store history of SupportingFiles i.e. each snapshot of a snippet has its
// own set of SupportingFiles.
type SupportingFile struct {
	ID int64 `json:"-"`

	SnapshotID int64 `json:"-"` // foreign key to CuratedSnippet table

	Path     string `json:"path"` // unique identifier for a file
	Contents []byte `json:"contents"`

	CreatedAt time.Time `json:"-"`
}

// PackageAccess records the time at which a user accessed the authoring interface for a particular package
type PackageAccess struct {
	Package   string `sql:"type:varchar(255) NOT NULL PRIMARY KEY" gorm:"column:Package"` // because GORM does not support string type primary keys if not specified via sql tag (https://github.com/jinzhu/gorm/issues/402)
	User      string `gorm:"column:User"`
	Timestamp int64  `gorm:"column:Timestamp"` // in seconds since epoch, as per time.Now().Unix()
}

// TableName tells GORM to use the returned string as the name of the table backing
// PackageAccess instead of following GORM table name conventions.
func (p PackageAccess) TableName() string {
	return "PackageAccess"
}

// A Run represents the output from executing a CuratedSnippet
type Run struct {
	ID           int64  `gorm:"column:ID;primary_key"`              // primary key
	SnippetID    int64  `sql:"not null" gorm:"column:SnippetID"`    // ID of snippet that was run
	Timestamp    int64  `sql:"not null" gorm:"column:Timestamp"`    // time when the code example was executed
	Stdin        []byte `gorm:"column:Stdin"`                       // byte stream that was sent to stdin
	Stdout       []byte `gorm:"column:Stdout"`                      // byte stream received on stdout
	Stderr       []byte `gorm:"column:Stderr"`                      // byte stream received on stderr
	Succeeded    bool   `sql:"not null" gorm:"column:Succeeded"`    // whether the run was successful
	SandboxError string `sql:"not null" gorm:"column:SandboxError"` // if Succeeded=false, the reason for failure (e.g. timeout, too much output)
}

// TableName tells GORM to use the returned string as the name of the table backing
// Run instead of following GORM table name conventions.
func (p Run) TableName() string {
	return "Run"
}

// An HTTPOutput represents an HTTP request sent to the code example together with the response
type HTTPOutput struct {
	ID                 int64  `gorm:"column:ID;primary_key"`                    // prinary key
	RunID              int64  `sql:"not null" gorm:"column:RunID"`              // ID of the run that generated this output
	RequestMethod      string `sql:"not null" gorm:"column:RequestMethod"`      // "GET", "POST", etc
	RequestURL         string `sql:"not null" gorm:"column:RequestURL"`         // URL in the request sent to the code example
	RequestHeaders     string `sql:"type:text" gorm:"column:RequestHeaders"`    // "key: value" pairs, separated by newlines
	RequestBody        []byte `gorm:"column:RequestBody"`                       // body of request sent to the code example
	ResponseStatus     string `sql:"not null" gorm:"column:ResponseStatus"`     // status message from response
	ResponseStatusCode int    `sql:"not null" gorm:"column:ResponseStatusCode"` // status code from response
	ResponseHeaders    string `sql:"type:text" gorm:"column:ResponseHeaders"`   // "key: value" pairs, separated by newlines
	ResponseBody       []byte `gorm:"column:ResponseBody"`                      // body of response received from code example
}

// TableName tells GORM to use the returned string as the name of the table backing
// HTTPOutput instead of following GORM table name conventions.
func (p HTTPOutput) TableName() string {
	return "HTTPOutput"
}

// An OutputFile represents a file that was created by a code example
type OutputFile struct {
	ID          int64  `gorm:"column:ID;primary_key"`             // primary key
	RunID       int64  `sql:"not null" gorm:"column:RunID"`       // ID of the run that generated this output
	Path        string `sql:"not null" gorm:"column:Path"`        // path relative to working directory in which code example ran
	ContentType string `sql:"not null" gorm:"column:ContentType"` // MIME type for file
	Contents    []byte `gorm:"column:Contents"`                   // contents of file
}

// TableName tells GORM to use the returned string as the name of the table backing
// OutputFile instead of following GORM table name conventions.
func (p OutputFile) TableName() string {
	return "OutputFile"
}

// A CodeProblem represents an error or warning encountered while building, running, or
// linting a code example. Examples include pylint violations and python runtime errors.
type CodeProblem struct {
	ID      int64  `gorm:"column:ID;primary_key"`          // primary key
	RunID   int64  `sql:"not null" gorm:"column:RunID"`    // ID of the run that generated this annotation
	Level   string `sql:"not null" gorm:"column:Level"`    // "info", "warning", or "error"
	Segment string `sql:"not null" gorm:"column:Segment"`  // "prelude", "code", or "postlude"
	Message string `sql:"type:text" gorm:"column:Message"` // a description of the issue
	Line    int    `sql:"not null" gorm:"column:Line"`     // line number (zero-based)
}

// TableName tells GORM to use the returned string as the name of the table backing
// CodeProblem instead of following GORM table name conventions.
func (p CodeProblem) TableName() string {
	return "CodeProblem"
}

// A Segment represents a chunk of code or output.
type Segment struct {
	ID     int64  `gorm:"column:ID;primary_key" json:"id"`                      // ID
	RunID  int64  `sql:"not null" gorm:"column:RunID" json:"run_id"`            // ID of associated run
	Type   string `sql:"not null" gorm:"column:Type" json:"type"`               // a segment type
	Region string `sql:"not null" gorm:"column:Region" json:"region,omitempty"` // "prelude", "main", or "postlude"

	Code            string      `sql:"not null" gorm:"column:Code" json:"code,omitempty"` // a chunk of source code
	BeginLineNumber int         `sql:"-" json:"begin_line_number"`                        // line of code (0-indexed) which begins this segment
	EndLineNumber   int         `sql:"-" json:"end_line_number"`                          // line of code (0-indexed) which ends this segment
	References      []Reference `sql:"-" json:"references,omitepty"`                      // list of references in this code segment

	Expression string `sql:"not null" gorm:"column:Expression" json:"expression,omitempty"` // expression string
	Value      string `sql:"not null" gorm:"column:Value" json:"value,omitempty"`           // value of the expression

	ImagePath     string `sql:"not null" gorm:"column:ImagePath" json:"image_path,omitempty"`         // path to an image
	ImageData     []byte `sql:"not null" gorm:"column:ImageData" json:"image_data,omitempty"`         // contents of the image
	ImageEncoding string `sql:"not null" gorm:"column:ImageEncoding" json:"image_encoding,omitempty"` // MIME type for an image
	ImageCaption  string `sql:"type:text" gorm:"column:ImageCaption" json:"image_caption,omitempty"`  // descriptive caption

	FilePath    string `sql:"not null" gorm:"column:FilePath" json:"file_path,omitempty"`        // path to a file
	FileData    []byte `sql:"not null" gorm:"column:FileData" json:"file_data,omitempty"`        // contents of the file
	FileCaption string `sql:"type:text" gorm:"column:FileCaption" json:"file_caption,omitempty"` // descriptive caption

	DirTablePath    string              `sql:"not null" gorm:"column:DirTablePath" json:"dir_table_path,omitempty"`        // path represented by the dir table
	DirTableCaption string              `sql:"type:text" gorm:"column:DirTableCaption" json:"dir_table_caption,omitempty"` // descriptive caption
	DirTableCols    []string            `sql:"not null" gorm:"column:DirTableCols" json:"dir_table_cols,omitempty"`
	DirTableEntries []annotate.DirEntry `sql:"not null" gorm:"column:DirTableEntries" json:"dir_table_entries,omitempty"` // info about directory entries

	DirTreePath    string            `sql:"not null" gorm:"column:DirTreePath" json:"dir_tree_path,omitempty"`
	DirTreeCaption string            `sql:"type:text" gorm:"column:DirTreeCaption" json:"dir_tree_caption,omitempty"`
	DirTreeEntries map[string]string `sql:"not null" gorm:"column:DirTreeEntries" json:"dir_tree_entries,omitempty"`
}

// TableName tells GORM to use the returned string as the name of the table backing
// Segment instead of following GORM table name conventions.
func (p Segment) TableName() string {
	return "Segment"
}

// --

// RelatedExamples contains the id a code example and the ids of
// its related code examples.
type RelatedExamples struct {
	SnippetID int64   `json:"snippet_id"`
	Examples  []int64 `json:"examples"`
}

// --

// OpenCodeExampleDb opens the central code example database, constructs a db map, and create any
// tables that do not already exist.
func OpenCodeExampleDb(db *sql.DB, dialect gorp.Dialect) *gorp.DbMap {
	dbmap := gorp.DbMap{
		Db:      db,
		Dialect: dialect,
	}
	dbmap.AddTable(CuratedSnippet{}).SetKeys(true, "SnapshotID")
	dbmap.AddTable(Comment{}).SetKeys(true, "ID")
	dbmap.AddTable(PackageAccess{}).SetKeys(false, "Package")
	dbmap.AddTable(Run{}).SetKeys(true, "ID")
	dbmap.AddTable(HTTPOutput{}).SetKeys(true, "ID")
	dbmap.AddTable(OutputFile{}).SetKeys(true, "ID")
	dbmap.AddTable(CodeProblem{}).SetKeys(true, "ID")
	dbmap.AddTable(Segment{}).SetKeys(true, "ID")

	// Create tables if they do not already exist
	return &dbmap
}
