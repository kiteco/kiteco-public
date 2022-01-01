package curation

import (
	"database/sql"

	gorp "gopkg.in/gorp.v1"
)

// FunctionCluster represents a cluster of function usages.
type FunctionCluster struct {
	FullIdent   string  `json:"fullIdentifier"` // FullIdent is the full function qualifier
	NumClusters int     `json:"numClusters"`    // NumClusters specifies the number of clusters used in the k-means clustering alg.
	ID          int     `json:"clusterID"`      // ID is the cluster id.
	Size        int     `json:"size"`           // Size is the number of snippets in the cluster.
	Percentage  float64 `json:"percentage"`     // Percentage is the ratio of snippets that are in this cluster>
	Code        []byte  `json:"code"`           // Code is the code of the representative snippet.
	Statement   []byte  `json:"statement"`      // Statement is the statement of the representative snippet.
}

// FunctionSnippet represents a code block that is presented to the user in a cluster.
type FunctionSnippet struct {
	FullIdent   string `json:"fullIdentifier"` // FullIdent is the full function qualifier.
	NumClusters int    `json:"numClusters"`    // NumClusters specifies the number of clusters used in the k-means clustering alg.
	ClusterID   int    `json:"clusterID"`      // ClusterID is the cluster id.
	ID          int    `json:"snippetID"`
	Code        []byte `json:"code"`      // Code is the content of the snippet.
	Statement   []byte `json:"statement"` // Statement is the line of code that contains the full function qualifier.
	Starred     bool   `json:"starred"`   // Starred keeps track whether a note is starred
}

// GithubCluster represents a cluster of code snippets.
type GithubCluster struct {
	FullIdent   string  `json:"fullIdentifier"` // FullIdent is the full function qualifier
	NumClusters int     `json:"numClusters"`    // NumClusters specifies the number of clusters used in the k-means clustering alg.
	ID          int     `json:"clusterID"`      // ID is the cluster id.
	X           float64 `json:"x"`              // X specifies the x-position of the embedding of this cluster.
	Y           float64 `json:"y"`              // Y specifies the x-position of the embedding of this cluster.
	Size        int     `json:"size"`           // Size is the number of snippets in the cluster.
	Percentage  float64 `json:"percentage"`     // Percentage is the ratio of snippets that are in this cluster>
	Code        []byte  `json:"code"`           // Code is the code of the representative snippet.
	Statement   []byte  `json:"statement"`      // Statement is the statement of the representative snippet.
}

// GithubSnippet represents a code block that is presented to the user in a cluster.
type GithubSnippet struct {
	FullIdent   string `json:"fullIdentifier"` // FullIdent is the full function qualifier.
	NumClusters int    `json:"numClusters"`    // NumClusters specifies the number of clusters used in the k-means clustering alg.
	ClusterID   int    `json:"clusterID"`      // ClusterID is the cluster id.
	ID          int    `json:"snippetID"`
	Code        []byte `json:"code"`      // Code is the content of the snippet.
	Statement   []byte `json:"statement"` // Statement is the line of code that contains the full function qualifier.
	Starred     bool   `json:"starred"`   // Starred keeps track whether a note is starred
}

// Note represents a note taken by the curator for a code snippet.
type Note struct {
	UniqueID string `json:"uniqueID"` // UniqueID of a snippet
	Content  string `json:"content"`  // content of the note
	Code     string `json:"code"`     // code of the snippet that the note is created for
}

// OpenGithubClustersDb opens the database that contains clustering
// results of github code snippets, constructs a db map,
// and create any tables that do not already exist.
func OpenGithubClustersDb(db *sql.DB, dialect gorp.Dialect) (*gorp.DbMap, error) {
	// Note that dbmap below is a global
	dbmap := gorp.DbMap{
		Db:      db,
		Dialect: dialect,
	}

	dbmap.AddTable(GithubCluster{}).SetKeys(false, "FullIdent", "NumClusters", "ID")
	dbmap.AddTable(GithubSnippet{}).SetKeys(false, "FullIdent", "NumClusters", "ClusterID", "ID")
	dbmap.AddTable(FunctionCluster{}).SetKeys(false, "FullIdent", "NumClusters", "ID")
	dbmap.AddTable(FunctionSnippet{}).SetKeys(false, "FullIdent", "NumClusters", "ClusterID", "ID")

	// Create tables if they do not already exist
	return &dbmap, nil
}
