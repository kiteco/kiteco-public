// Package kitelocal re-exports useful pieces of the internal package kitelocal for testing, etc
package kitelocal

import (
	"context"

	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// LoadOptions specifies various loading options for python services
type LoadOptions = kitelocal.LoadOptions

// LoadResourceManager loads the resource manager for Kite Local
func LoadResourceManager(ctx context.Context, opts LoadOptions) (pythonresource.Manager, error) {
	return kitelocal.LoadResourceManager(ctx, opts)
}
