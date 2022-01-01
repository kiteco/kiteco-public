//go:generate go-bindata -pkg manifest -prefix ../.. ../../manifest.json

package manifest

import (
	"bytes"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// KiteManifest is the global read-only manifest for all Kite resources
var KiteManifest Manifest

type emptyName struct {
	*bytes.Reader
}

func (r emptyName) Name() string { return "" }

func init() {
	m, err := New(emptyName{bytes.NewReader(MustAsset("manifest.json"))})
	if err != nil {
		panic(err)
	}

	KiteManifest = m

	if _, ok := KiteManifest[keytypes.BuiltinDistribution3]; !ok {
		panic("Python 3 builtin distribution not found in KiteManifest")
	}
}
