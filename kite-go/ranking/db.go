package ranking

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-golib/hash"
	gorp "gopkg.in/gorp.v1"
)

const (
	// Classification for ranking results for the active learning
	// data collection interface.

	// GoodResult indicates that the ranking result for a query is good.
	GoodResult int = 1 << iota

	// BadResult indicates that the ranking result for a query is bad.
	BadResult

	// UnclearQuery indicates that the query is ambiguous.
	UnclearQuery

	// NoExamples indicates that there are not examples for the query.
	NoExamples
)

// QueryType defines query types. A query can be either an active one
// or a passive one.
type QueryType int

const (
	// Active indicates the query is an active search query.
	Active QueryType = iota // 0

	// Passive indicates that the query is a passive search query.
	Passive // 1
)

// QueryManager is the interface that interacts with the DB.
type QueryManager struct {
	db gorm.DB
}

// NewQueryManager returns a new QueryManager.
func NewQueryManager(db gorm.DB) *QueryManager {
	return &QueryManager{db: db}
}

// InsertDocQuery creates a new record in the query table.
func (r *QueryManager) InsertDocQuery(query *DocQuery) error {
	if err := r.db.Create(query).Error; err != nil {
		return fmt.Errorf("error inserting new query: %v", err)
	}
	return nil
}

// InsertQuery creates a new record in the query table.
func (r *QueryManager) InsertQuery(query *Query) error {
	if err := r.db.Create(query).Error; err != nil {
		return fmt.Errorf("error inserting new query: %v", err)
	}
	return nil
}

// GetAllDocQueries returns all doc queries in the doc query table.
func (r *QueryManager) GetAllDocQueries() ([]*DocQuery, error) {
	var queries []*DocQuery
	err := r.db.Find(&queries).Error
	return queries, err
}

// GetAllQueries returns all queries in the query table.
func (r *QueryManager) GetAllQueries() ([]*Query, error) {
	var queries []*Query
	err := r.db.Find(&queries).Error
	return queries, err
}

// AddDocLabel creates a new record in the ranking table.
func (r *QueryManager) AddDocLabel(label *DocLabel) error {
	if err := r.db.Create(label).Error; err != nil {
		return fmt.Errorf("error inserting new ranking: %v", err)
	}
	return nil
}

// AddLabel creates a new record in the ranking table.
func (r *QueryManager) AddLabel(label *Label) error {
	if err := r.db.Create(label).Error; err != nil {
		return fmt.Errorf("error inserting new ranking: %v", err)
	}
	return nil
}

// GetAllLabels returns all the records in the label table.
func (r *QueryManager) GetAllLabels() ([]Label, error) {
	var labels []Label
	err := r.db.Find(&labels).Error
	return labels, err
}

// SelectLabelsByQueryID returns all the records in the label table.
func (r *QueryManager) SelectLabelsByQueryID(id uint64) ([]Label, error) {
	var labels []Label
	err := r.db.Where("query_id = ?", id).Find(&labels).Error
	return labels, err
}

// SelectQuery returns the specific query entry with the requested id in the query table.
func (r *QueryManager) SelectQuery(id uint64) (*Query, error) {
	var query Query
	err := r.db.Find(&query, id).Error
	if err != nil {
		switch err {
		case gorm.RecordNotFound:
			return nil, fmt.Errorf("cannot find query with ID=%d", id)
		default:
			return nil, fmt.Errorf("error selecting query with ID=%d: %v", id, err)
		}
	}
	return &query, nil
}

// SelectQueryByType returns the query entries of the requested type.
func (r *QueryManager) SelectQueryByType(typ QueryType) ([]*Query, error) {
	var queries []*Query
	err := r.db.Where("type = ?", typ).Find(&queries).Error
	return queries, err
}

// SelectQueryByText returns the query entries with the requested text.
func (r *QueryManager) SelectQueryByText(text string) ([]*Query, error) {
	var queries []*Query
	err := r.db.Where("text = ?", text).Find(&queries).Error
	return queries, err
}

// SelectDocQueryByText returns the query entries with the requested text.
func (r *QueryManager) SelectDocQueryByText(text string) ([]*DocQuery, error) {
	var queries []*DocQuery
	err := r.db.Where("text = ?", text).Find(&queries).Error
	return queries, err
}

// Migrate checks whether the db has the right tables.
func (r *QueryManager) Migrate() error {
	if err := r.db.AutoMigrate(&Query{}, &Label{}, &DocQuery{}, &DocLabel{}).Error; err != nil {
		return fmt.Errorf("error creating tables in db: %v", err)
	}
	return nil
}

// Hash returns the hash value of a query.
func (q *Query) Hash() string {
	return hash.SpookyHash128String(append(q.Code, []byte(q.Text)...))
}

// DocQuery represents a query for a doc ranking task.
type DocQuery struct {
	ID        uint64
	Timestamp int64
	User      string
	Text      string
	Eval      int
	Package   string
}

// DocLabel represents a piece of documentation and its relevant to the query.
type DocLabel struct {
	ID        uint64
	QueryID   uint64
	Signature string
	Rank      float64
}

// Query represents the query, either a chunk of code and a cursor position or a string or both, used in the web ranking UI.
type Query struct {
	ID        uint64
	Timestamp int64  // timestamp in epoch seconds at which the user submitted these results
	User      string // name of user

	Text         string // text query
	Path         string // path to file containing this query
	Code         []byte // the full source file from which this example was drawn
	BeginLine    int    // first line presented to user
	EndLine      int    // last line presented to user
	CursorLine   int    // line number of cursor
	CursorColumn int    // column number of cursor (= bytes from beginning of line)
	CursorOffset int    // cursor offset in bytes from beginning of file
	CursorIndex  int    // index of this query within the source file
	Package      string
	Type         QueryType // query type
	Eval         int       // evaluation of this query. The value should be one of (GoodResult, BadResult, NoExamples, UnclearQuery)
}

// Label represents a code example with a relevance score that was assigned through the web interface for a specific query.
type Label struct {
	ID         uint64
	QueryID    uint64
	Hash       string  // value of snippet.Hash().String()
	Code       []byte  // the code example content
	Path       string  // path to source file
	Rank       float64 // relevance score of this code example (zero means the least relevant)
	BeginLine  int     // index within the file of the first line in this code example
	EndLine    int     // index within the file of the last line in this code example
	SnippetID  int64
	SnapshotID int64
	Corpus     string // corpus from which this candidate was generated
	Supervised bool   // a flag that denotes whether a labelled is supervised. If it's supervised, then it's value will over write all other labels for the same query-example pairs.
}

// OpenQueryDb opens an sqlite database, constructs a db map, and create any tables that do
// not already exist.
func OpenQueryDb(path string) (*gorp.DbMap, error) {
	var err error // necessary because db is a global
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	// Note that dbmap below is a global
	dbmap := gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	dbmap.AddTable(Query{}).SetKeys(true, "ID")
	dbmap.AddTable(Label{}).SetKeys(false, "QueryID", "Hash")

	// Create tables if they do not already exist
	return &dbmap, nil
}
