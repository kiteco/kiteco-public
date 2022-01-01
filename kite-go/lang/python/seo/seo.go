package seo

import (
	"compress/gzip"
	"encoding/gob"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

const (
	// DefaultDataPath is where the latest SEO data is stored
	DefaultDataPath = "s3://kite-data/seo/20190808.gob.gz"
)

// Data encapsulates SEO-specific data
type Data map[keytypes.Distribution]map[pythonimports.Hash]pythonimports.DottedPath

// CanonicalLinkPath returns the <link rel="canonical"> Symbol path for the given Symbol's web docs page.
// This may differ from the internal canonical path for the Symbol.
// If an empty path is returned, the web docs page should not be indexed (noindex, nofollow).
func (d Data) CanonicalLinkPath(sym pythonresource.Symbol) pythonimports.DottedPath {
	sym = sym.Canonical()
	return d[sym.Distribution()][sym.PathHash()]
}

// IterateCanonicalLinkPaths iterates via a callback
func (d Data) IterateCanonicalLinkPaths(cb func(pythonimports.DottedPath) bool) {
	for _, m := range d {
		for _, p := range m {
			if !cb(p) {
				return
			}
		}
	}
}

// Encode encodes Data to a writer
func (d Data) Encode(w io.Writer) (err error) {
	gw := gzip.NewWriter(w)
	defer deferErr(&err, gw.Close)
	return gob.NewEncoder(gw).Encode(d)
}

// Decode loads from a writer
func (d *Data) Decode(r io.Reader) (err error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer deferErr(&err, gr.Close)
	return gob.NewDecoder(gr).Decode(d)
}

// Load loads data from the given path (s3 or local)
func Load(path string) (Data, error) {
	rc, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var d Data
	d.Decode(rc)
	return d, nil
}

func deferErr(err *error, f func() error) {
	if e := f(); *err == nil {
		*err = e
	}
}
