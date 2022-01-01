package permissions

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
)

func Test_Component(t *testing.T) {
	m := requireManager(t)

	component.TestImplements(t, m, component.Implements{
		Handlers: true,
	})
}

func TestName(t *testing.T) {
	m := requireManager(t)
	assert.Equal(t, "permissions", m.Name())
}

func TestAuthorizedEmptyFile(t *testing.T) {
	m := requireManager(t)

	reason, authorized, err := m.Authorized("")
	assert.NoError(t, err)
	assert.False(t, authorized)
	assert.Equal(t, "language not supported", reason.String())
}

// -

func requireManager(t *testing.T) *Manager {
	return NewTestManager(lang.Python)
}
