package helpers

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	// ThirdParty2StubKey for Python 2 third party libs
	ThirdParty2StubKey = "stub_thirdparty2"
	// ThirdParty3StubKey for Python 3 third party libs
	ThirdParty3StubKey = "stub_thirdparty3"
	// Builtin3StubKey for Python 3 builtin standard libs
	Builtin3StubKey = "stub_builtin3"
)

// LoadAnalyzed loads analyzed SourceTrees keyed by Distribution
func LoadAnalyzed(path string, graph pythonresource.Manager) (map[string]*pythonenv.SourceTree, error) {
	if path == "" {
		return nil, nil
	}

	fileR, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	defer fileR.Close()

	jsonR, err := gzip.NewReader(fileR)
	if err != nil {
		return nil, err
	}
	defer jsonR.Close()

	var flats map[string]pythonenv.FlatSourceTree
	err = json.NewDecoder(jsonR).Decode(&flats)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*pythonenv.SourceTree)
	for key, flat := range flats {
		st, err := flat.Inflate(graph)
		if err != nil {
			return nil, err
		}
		out[key] = st
	}
	return out, nil
}

// StoreAnalyzed stores analyzed SourceTrees keyed by Distribution
func StoreAnalyzed(path string, stubs map[string]*pythonenv.SourceTree) error {
	flats := make(map[string]pythonenv.FlatSourceTree)
	for key, st := range stubs {
		flat, err := st.Flatten(kitectx.Background())
		if err != nil {
			return err
		}
		flats[key] = *flat
	}

	fileW, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fileW.Close()

	jsonW := gzip.NewWriter(fileW)
	defer jsonW.Close()

	return json.NewEncoder(jsonW).Encode(flats)
}

// QualifiedPathAnalysis returns a "fully qualified" dotted path for the value corresponding to the given address from offline analysis.
func QualifiedPathAnalysis(a pythontype.Address) (pythonimports.DottedPath, error) {
	if a.File == "" {
		return a.Path, nil
	}

	rel, err := filepath.Rel("/", a.File)
	if err != nil {
		return pythonimports.DottedPath{}, err
	}

	fileParts := strings.Split(strings.TrimSuffix(rel, ".py"), "/")
	if len(fileParts) > 0 {
		switch fileParts[len(fileParts)-1] {
		case "__init__", "": // package path: toss that last component
			fileParts = fileParts[:len(fileParts)-1]
		}
	}

	return pythonimports.NewPath(append(fileParts, a.Path.Parts...)...), nil
}

// PseudoFilePath returns the pseudo file path to use for analysis for the given root (e.g. /.../typeshed/stdlib/3) and full file path.
func PseudoFilePath(root, path string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}

	ext := filepath.Ext(rel)
	switch ext {
	case ".pyi":
		rel = strings.TrimSuffix(rel, ".pyi") + ".py"
	case ".py":
	default:
		return "", errors.Errorf("invalid extension %s for analysis file", ext)
	}
	return "/" + rel, nil
}
