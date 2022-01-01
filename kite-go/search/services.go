package search

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/localcode"
)

// Options contains all the options structures required to load up search services
type Options struct {
	PythonOptions *python.ServiceOptions
}

// Services wraps all services for active and passive search
type Services struct {
	Python *python.Services
	Local  *localcode.Client
}

// ServicesHandler wraps a Services object and provides HTTP access to the
// underlying functionality
type ServicesHandler struct {
	services *Services
	Python   *python.ServicesHandler
}

// NewServices builds Services from the given Options.
func NewServices(opts *Options) (*Services, error) {
	local := localcode.NewClient()

	python, err := python.NewServices(opts.PythonOptions, local)
	if err != nil {
		return nil, fmt.Errorf("error loading python services: %s", err)
	}

	// TODO(tarak): Other components should return errors on failure instead
	// of silently continuing.
	return &Services{
		Python: python,
		Local:  local,
	}, nil
}

// MockServices builds a services object containing mock data.
func MockServices(t testing.TB, opts *Options) (*Services, error) {
	mockNodes := []string{
		"os.path.join",
		"json.dump",
		"json.dumps",
		"json.load",
		"json.loads",
		"numpy.zero",
		"numpy.linalg.dot",
	}
	python := python.MockServices(t, opts.PythonOptions, mockNodes...)

	// TODO(tarak): Other components should return errors on failure instead
	// of silently continuing.
	return &Services{
		Python: python,
	}, nil
}

// NewServicesHandler builds ServicesHandlers from Services objects
func NewServicesHandler(services *Services) (*ServicesHandler, error) {
	if services == nil {
		return nil, fmt.Errorf("error building handler with null services")
	}

	python, err := python.NewServicesHandler(services.Python)
	if err != nil {
		return nil, err
	}

	return &ServicesHandler{
		services: services,
		Python:   python,
	}, nil
}

// --
