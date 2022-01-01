package signatures

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/permissions"
	"github.com/kiteco/kiteco/kite-go/lang"
)

func Test_Component(t *testing.T) {
	m := &Manager{}
	component.TestImplements(t, m, component.Implements{
		Initializer: true,
		Handlers:    true,
	})
}

func Test_Signatures(t *testing.T) {
	_, _, server := setupManager()
	defer server.Close()
}

// --

func setupManager() (*driver.TestProvider, component.PermissionsManager, *httptest.Server) {
	provider := driver.NewTestProvider()
	m := NewManager(provider, Options{})

	f := filepath.Join(os.TempDir(), "test_permissions.json")
	os.RemoveAll(f)

	p := permissions.NewTestManager(lang.Python)
	m.Initialize(component.InitializerOptions{
		Permissions: p,
	})
	m.cohort = &component.MockCohortManager{}

	mux := mux.NewRouter()
	m.RegisterHandlers(mux)

	ts := httptest.NewServer(mux)
	return provider, p, ts
}
