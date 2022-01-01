package curation

import (
	"database/sql"

	gorp "gopkg.in/gorp.v1"
)

// DiffQuery represents the query sent to the cluster labeling web interface. Since each query
// has an unique ID, we encodes each cluster query by their ID.
type DiffQuery struct {
	// We store ID using string instead of uint64 because sql can't handle large uint64 values.
	// See http://go-database-sql.org/surprises.html
	ID        string // ID of the query, which is the hash value of the error message.
	Timestamp int64  // timestamp in epoch seconds at which the user submitted these results
	User      string // name of user
	Message   string // error message of the query
}

// CodeQuery represents the query sent to the cluster labeling web interface.
type CodeQuery struct {
	ID                 string // ID of the query.
	Timestamp          int64  // timestamp in epoch seconds at which the user submitted these results
	User               string // name of user
	Token              string // token unser the cursor
	Path               string // path to the file from which the query is extracted from
	Code               []byte // source code
	BeginLine          int    // first line presented to user
	EndLine            int    // end line presented to user
	Window             []byte // lines that were presented to the user
	CursorIndex        int    // index of this cursor within the file
	WindowCursorOffset int    // offset of cursor within snippet
}

// Cluster stores the meta data of a cluster
type Cluster struct {
	QueryID   string // ID of the original query
	ClusterID int    // ID of the cluster should be greater or equal to 0
	Notes     string // Description of the cluster
	MemberNum int    // Number of members
}

// EpisodeLabel encodes the cluster ID of an episode
type EpisodeLabel struct {
	QueryID   string // ID of the original query
	EpisodeID string // ID of the episode
	ClusterID int    // ID of the cluster that the episode belongs to
	Text1     string // text of the first file of the diff
	Text2     string // text of the second file of the diff
	Tag       string // tag of the episode
}

// CodeLabel encodes the cluster ID of an code example candidate
type CodeLabel struct {
	QueryID   string // ID of the original query
	CodeID    string // ID of the code example
	ClusterID int    // ID of the cluster that the episode belongs to
	Hash      string // hash of the code
	Code      []byte // source code
	Path      string // path to the file from which candidate is extracted
}

// OpenClusterDb opens an sqlite database, constructs a db map, and create any tables that do
// not already exist.
func OpenClusterDb(db *sql.DB, dialect gorp.Dialect) (*gorp.DbMap, error) {
	dbmap := gorp.DbMap{
		Db:      db,
		Dialect: dialect,
	}

	dbmap.AddTable(DiffQuery{}).SetKeys(false, "ID")
	dbmap.AddTable(CodeQuery{}).SetKeys(false, "ID")

	dbmap.AddTable(Cluster{}).SetKeys(false, "QueryID", "ClusterID")

	dbmap.AddTable(EpisodeLabel{}).SetKeys(false, "EpisodeID", "QueryID", "ClusterID")
	dbmap.AddTable(CodeLabel{}).SetKeys(false, "CodeID", "QueryID", "ClusterID")

	// Create tables if they do not already exist
	return &dbmap, nil
}
