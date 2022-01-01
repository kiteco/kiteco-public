package stackoverflow

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// DefaultOptions contains default options for the stackoverflow index
var DefaultOptions = Options{
	Path: "s3://kite-data/stackoverflow/2017-03-09_01-15-22-PM/index.json.diskmap",
}

// Options contains options for the stackoverflow index
type Options struct {
	Path string
}

// Result is a stackoverflow post returned in response to a user query.
type Result struct {
	Title  string
	PostID int64
	Score  int64
}

// ResultSet is the top-level struct returned in response to queries.
type ResultSet struct {
	Results []*Result
}

// Index represents a set of stackoverflow posts indexed by graph node
type Index struct {
	graph *pythonimports.Graph
	index *diskmap.Map
}

// Load loads a stackoverflow index
func Load(graph *pythonimports.Graph, opts Options) (*Index, error) {
	// download the diskmap to a local path
	localpath, err := fileutil.DownloadedFile(opts.Path)
	if err != nil {
		return nil, err
	}

	// load the diskmap
	index, err := diskmap.NewMap(localpath)
	if err != nil {
		return nil, err
	}

	return &Index{
		graph: graph,
		index: index,
	}, nil
}

// LookupNode gets a list of stackoverflow posts relevant to a given node
func (i *Index) LookupNode(node *pythonimports.Node) ([]*Result, error) {
	anypath, found := i.graph.AnyPaths[node]
	if !found {
		return nil, nil
	}

	// now query the index using the anypath for this node
	var r ResultSet
	err := diskmap.JSON.Get(i.index, anypath.String(), &r)
	if err != nil {
		return nil, fmt.Errorf("error looking up key %s: %v", anypath.String(), err)
	}

	return r.Results, nil
}
