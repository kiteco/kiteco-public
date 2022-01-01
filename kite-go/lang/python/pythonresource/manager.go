package pythonresource

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/stringutil"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/toplevel"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

// Options encapsulates arguments to NewManager; most clients should use DefaultOptions
// Note: if Dists is nil, all distributions mentioned in Manifest will be loaded;
//       pass an empty slice to load no distributions
type Options struct {
	Manifest     manifest.Manifest
	Dists        []keytypes.Distribution
	DistIndex    distidx.Index
	ToplevelPath string

	DistLoadAttempts int
	DistLoadBackoff  time.Duration

	// Concurrency indicates the number of goroutines to use for loading; a zero value is replaced with 32
	Concurrency int

	// CacheSize indicates the number of packages we are allowed to dynamically loaded
	CacheSize int

	// DisableDynamicLoading will prevent new distributions from being loaded once initialzed via Dists
	DisableDynamicLoading bool
}

// SymbolOnly returns the same options but with a manifest that only contains the symbol graph.
func (o Options) SymbolOnly() Options {
	o.Manifest = o.Manifest.SymbolOnly()
	return o
}

// WithCustomPaths loads a manifest and/or distidx from custom (local) paths.
// If the provided paths are empty, the input is unchanged.
func (o Options) WithCustomPaths(manifestPath, distidxPath string) (Options, error) {
	if manifestPath != "" {
		mF, err := os.Open(manifestPath)
		if err != nil {
			return Options{}, errors.Wrapf(err, "could not open manifest path %s", manifestPath)
		}
		defer mF.Close()
		o.Manifest, err = manifest.New(mF)
		if err != nil {
			return Options{}, errors.Wrapf(err, "could not load manifest from %s", manifestPath)
		}
	}
	if distidxPath != "" {
		dF, err := os.Open(distidxPath)
		if err != nil {
			return Options{}, errors.Wrapf(err, "could not open distidx path %s", distidxPath)
		}
		defer dF.Close()
		o.DistIndex, err = distidx.New(dF)
		if err != nil {
			return Options{}, errors.Wrapf(err, "could not load distidx from %s", distidxPath)
		}
	}
	return o, nil
}

// DefaultOptions is the default argument to NewManager
var DefaultOptions = Options{
	Manifest:  manifest.KiteManifest,
	Dists:     nil,
	DistIndex: distidx.KiteIndex,

	DistLoadAttempts: 3,
	DistLoadBackoff:  5 * time.Second,
}

// DefaultLocalOptions is the default argument to NewManager for Kite Local.
var DefaultLocalOptions = Options{
	Manifest:     manifest.KiteManifest,
	Dists:        []keytypes.Distribution{},
	DistIndex:    distidx.KiteIndex,
	ToplevelPath: "s3://kite-resource-manager/python-v2/toplevel20200720.gob.gz",

	DistLoadAttempts: 3,
	DistLoadBackoff:  5 * time.Second,
	Concurrency:      2,
	CacheSize:        50,
}

var smallDists = []keytypes.Distribution{
	keytypes.BuiltinDistribution3,
	keytypes.NumpyDistribution,
	keytypes.RequestsDistribution,
	keytypes.TensorflowDistribution,
	keytypes.BotoDistribution,
	keytypes.MatplotlibDistribution,
	keytypes.GoogleDistribution,
}

// SmallOptions is a small dataset for testing.
var SmallOptions = Options{
	Manifest:  manifest.KiteManifest,
	Dists:     smallDists,
	DistIndex: distidx.KiteIndex,

	DistLoadAttempts: 3,
	DistLoadBackoff:  5 * time.Second,
}

// Manager manages loading & unloading of datasets (resources) associated with Distributions
type manager struct {
	manifest manifest.Manifest
	index    distidx.Index
	cache    *lru.Cache
	toplevel toplevel.Entities

	loadErr  error
	loadLock sync.Mutex
}

// NewManager creates a new resource Manager;
// it is non-blocking: an error (or nil) is sent on the returned channel once loading is complete.
func NewManager(opts Options) (Manager, <-chan error) {
	return NewManagerWithCtx(context.Background(), opts)
}

// NewManagerWithCtx creates a new resource Manager;
// it is non-blocking: an error (or nil) is sent on the returned channel once loading is complete.
func NewManagerWithCtx(ctx context.Context, opts Options) (Manager, <-chan error) {
	if opts.Dists == nil {
		opts.Dists = opts.Manifest.Distributions()
	}
	if opts.DisableDynamicLoading {
		opts.Manifest = opts.Manifest.FilterDistributions(opts.Dists)
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 32
	}
	if opts.CacheSize == 0 {
		// Avoid eviction if cache size is not set by setting cache size to the
		// total number of distributions in the manifest
		opts.CacheSize = opts.Manifest.NumDistributions()
	}

	m := &manager{
		manifest: opts.Manifest,
		index:    opts.DistIndex,
	}

	c := make(chan error, 1)
	var err error
	m.cache, err = lru.New(opts.CacheSize)
	if err != nil {
		go func(err error) {
			c <- err
		}(err)
	}
	go m.unloader(ctx)

	go func() {
		m.loadErr = m.loadInitialDists(opts)
		c <- m.loadErr // non-blocking due to buffer
	}()

	return m, c
}

// Close releases all resources used by this manager
func (rm *manager) Close() error {
	return nil
}

// Distributions returns all available Distributions from the manifest for this Manager
func (rm *manager) Distributions() []keytypes.Distribution {
	return rm.manifest.Distributions()
}

// DistLoaded returns whether a given distribution is loaded
func (rm *manager) DistLoaded(dist keytypes.Distribution) bool {
	_, ok := rm.getCached(dist, false, "")
	return ok
}

// Reset clears the resource cache
func (rm *manager) Reset() {
	if rm.cache == nil {
		return
	}
	// If there's a race, it's better to clear stringutil first:
	//  stringutil.Clear() -> rm.loadResourceGroup(...) -> rm.cache.Purge() clear leaves extra stuff in stringutil, but the state is sound
	//  rm.cache.Purge() -> rm.loadResourceGroup(...) -> stringutil.Clear() causes resource data to reference stringutil data that no longer exists
	// The second case is not sound, since it'll cause panics when that data is accessed.
	stringutil.Clear()
	rm.cache.Purge()
}

// -

// resourceGroupLoadable checks if a resource group for dist is loadable (but not necessarily preloaded/cached)
func (rm *manager) resourceGroupLoadable(dist keytypes.Distribution) bool {
	_, ok := rm.manifest[dist]
	return ok
}

func (rm *manager) resourceGroup(dist keytypes.Distribution) *resources.Group {
	if dist.Name == "" { // in particular, ignore the zero Distribution
		return nil
	}

	if rg, ok := rm.getCached(dist, false, ""); ok {
		return rg
	}

	return nil
}

func (rm *manager) loadResourceGroup(dist keytypes.Distribution, src string) *resources.Group {
	if dist.Name == "" { // in particular, ignore the zero Distribution
		return nil
	}

	if rg, ok := rm.getCached(dist, true, src); ok {
		return rg
	}

	return nil
}

type dynamicDistribution struct {
	ResourceGroup *resources.Group
	LoadedAt      time.Time
	UsedAt        time.Time
	LoadedFrom    string
}

func (rm *manager) getCached(dist keytypes.Distribution, load bool, src string) (*resources.Group, bool) {
	if rm.cache == nil {
		return nil, false
	}

	// Serialize calls that may load new data
	if load {
		rm.loadLock.Lock()
		defer rm.loadLock.Unlock()
	}

	now := time.Now()
	obj, ok := rm.cache.Get(dist)
	switch {
	case ok:
		dd := obj.(dynamicDistribution)
		dd.UsedAt = now
		rm.cache.Add(dist, dd)
		return dd.ResourceGroup, true
	case !ok && load:
		rg, err := rm.manifest.Load(dist)
		if err != nil {
			log.Printf("error loading distribution %s: %s", dist, err)
			return nil, false
		}
		rm.cache.Add(dist, dynamicDistribution{
			ResourceGroup: rg,
			LoadedAt:      now,
			UsedAt:        now,
			LoadedFrom:    src,
		})

		log.Println("loaded", dist.String(), "in", time.Since(now))
		return rg, true
	}

	return nil, false
}

const (
	unloadCheckFrequency  = 1 * time.Minute
	unloadTimeoutDuration = 30 * time.Minute
)

func (rm *manager) unloader(ctx context.Context) {
	ticker := time.NewTicker(unloadCheckFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			keys := rm.cache.Keys()
			for _, key := range keys {
				obj, ok := rm.cache.Peek(key)
				if !ok {
					continue
				}
				dd := obj.(dynamicDistribution)
				if time.Since(dd.UsedAt) > unloadTimeoutDuration {
					rm.cache.Remove(key)
					log.Println("unloaded", key.(keytypes.Distribution).String(), "after", time.Since(dd.UsedAt))
				}
			}
		}
	}
}

func (rm *manager) loadInitialDists(opts Options) error {
	var concurrentLoad = func(dist keytypes.Distribution) error {
		rg, err := opts.Manifest.Load(dist)
		if err != nil {
			return errors.Wrapf(err, "error loading resources for distribution %s", dist)
		}
		now := time.Now()
		rm.cache.Add(dist, dynamicDistribution{
			ResourceGroup: rg,
			LoadedAt:      now,
			UsedAt:        now,
		})
		return nil
	}

	// load all the resources async
	var jobs []workerpool.Job
	for _, dist := range opts.Dists {
		d := dist // new variable to close over
		jobs = append(jobs, func() error {
			var err error
			for i := 0; i < opts.DistLoadAttempts; i++ {
				time.Sleep(time.Duration(int64(opts.DistLoadBackoff) * int64(i)))
				err = concurrentLoad(d)
				if err == nil {
					return nil
				}

				// TODO(naman): distinguish between retriable and fatal errors? possibly more work
				// than its worth, but just incase...
				log.Printf("error loading dist %s on attempt %d of %d: %s",
					d.String(), i+1, opts.DistLoadAttempts, err)
			}
			return err
		})
	}

	// also load toplevel entries async
	if opts.ToplevelPath != "" {
		jobs = append(jobs, func() error {
			var err error
			rm.toplevel, err = toplevel.Load(opts.ToplevelPath)
			return err
		})
	}

	pool := workerpool.New(opts.Concurrency)
	defer pool.Stop()
	pool.Add(jobs)
	if err := pool.Wait(); err != nil {
		return err
	}

	return nil
}
