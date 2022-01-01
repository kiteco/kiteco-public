package statusicon

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

func Test_Component(t *testing.T) {
	m := NewManager(nil)
	component.TestImplements(t, m, component.Implements{
		Initializer: true,
		Settings:    true,
		UserAuth:    true,
	})
}
