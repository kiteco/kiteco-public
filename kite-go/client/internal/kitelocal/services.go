package kitelocal

import (
	"context"
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
	"github.com/pkg/errors"
)

// LoadOptions specifies various loading options for python services
type LoadOptions struct {
	Blocking               bool                    // Waits for resource manager to finish loading all distributions
	DatadepsMode           bool                    // Will load all packages we know about
	Dists                  []keytypes.Distribution // Load custom distributions if defined, pass nil for default behavior (i.e dynamic)
	DisableDynamicLoading  bool                    // Will disable loading of additional distributions after initial load with Dists
	RemoteResourcesManager string                  // Will connect to a remote python resource manager instead of launch it locally
}

// LoadResourceManager loads the resource manager for Kite Local
func LoadResourceManager(ctx context.Context, opts LoadOptions) (pythonresource.Manager, error) {
	// DatadepsMode enables downloading of all required datasets, including the full
	// resource manager. We also have to block so that everything completes before we
	// generate the datadeps dataset.
	if opts.DatadepsMode {
		opts.Blocking = true
	}

	// Tweak resource manager options as needed
	rmOpts := pythonresource.DefaultLocalOptions
	rmOpts.DisableDynamicLoading = opts.DisableDynamicLoading

	var dists []keytypes.Distribution
	if !opts.DatadepsMode {
		dists = []keytypes.Distribution{} // Ensures no distributions are loaded initially (see pythonresource.Options)
	}

	switch {
	case opts.Dists != nil:
		// Used in tests. We set the cache size to zero here b/c of a subtlty with resource manager's
		// resourceGroupLoadable method.
		dists = opts.Dists
	case opts.DatadepsMode:
		// We can load data faster in datadeps mode
		rmOpts.Concurrency = 32
	}

	if opts.RemoteResourcesManager != "" {
		//fixme handle blocking?
		log.Println("Using remote resource manager at", opts.RemoteResourcesManager)
		mgr, err := pythonresource.NewRPCClient(opts.RemoteResourcesManager)
		if err != nil {
			return nil, err
		}
		return pythonresource.NewLoggingManager(pythonresource.NewCachingManager(pythonresource.NewLoggingManager(mgr, true, "INNER_LOGGER")), true, "OUTER_LOGGER"), nil
	}

	rmOpts.Dists = dists
	resourceManager, errc := pythonresource.NewManagerWithCtx(ctx, rmOpts)
	if opts.Blocking {
		if err := <-errc; err != nil {
			return nil, err
		}
	} else {
		go func() {
			if err := <-errc; err != nil {
				panic(err)
			}
		}()
	}
	return resourceManager, nil
}

// LoadPythonServices builds the python.Services object to be used by kite local.
func LoadPythonServices(ctx context.Context, opts LoadOptions) (*python.Services, error) {
	resourceManager, err := LoadResourceManager(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Make sure we force models to load so they appear in datadeps
	if opts.DatadepsMode {
		tensorflow.ForceLoadCycle = true
	}

	serviceOptions := python.DefaultServiceOptions
	serviceOptions.ModelOptions.Local = true

	models, err := pythonmodels.NewWithCtx(ctx, serviceOptions.ModelOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create models")
	}

	return &python.Services{
		Options:         &serviceOptions,
		ResourceManager: resourceManager,
		Models:          models,
	}, nil
}
