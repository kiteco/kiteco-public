package localcode

import (
	"io"
	"io/ioutil"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang"
)

var (
	loaderLock sync.Mutex
	loaders    = map[lang.Language]LoaderFunc{}
)

// RegisterLoader registers a loader function for a language
func RegisterLoader(l lang.Language, b LoaderFunc) {
	loaderLock.Lock()
	defer loaderLock.Unlock()
	loaders[l] = b
}

func getLoader(l lang.Language) (LoaderFunc, bool) {
	loaderLock.Lock()
	defer loaderLock.Unlock()
	b, ok := loaders[l]
	return b, ok
}

// Getter is an interface that can retrieve named artifacts
type Getter interface {
	Get(name string) ([]byte, error)
	GetReader(name string) (io.ReadCloser, error)
	ID() string
	List() []string
}

// Cleanuper is an interface that defines the ability to cleanup/remove any state
// associated with the object its implemented on.
type Cleanuper interface {
	Cleanup() error
}

// LoaderFunc is used to load artifacts into a user-specified object to be tracked by the localcode package.
type LoaderFunc func(getter Getter) (Cleanuper, error)

// --

type artifactGetter struct {
	client   *artifactClient
	artifact artifact
}

func newArtifactGetter(artifact artifact, client *artifactClient) *artifactGetter {
	return &artifactGetter{
		client:   client,
		artifact: artifact,
	}
}

// Get implements Getter
func (a *artifactGetter) Get(name string) ([]byte, error) {
	r, err := a.client.getReader(a.artifact, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	return buf, err
}

// GetReader implements Getter
func (a *artifactGetter) GetReader(name string) (io.ReadCloser, error) {
	return a.client.getReader(a.artifact, name)
}

// ID implements Getter
func (a *artifactGetter) ID() string {
	return a.artifact.UUID
}

// List implements Getter
func (a *artifactGetter) List() []string {
	return a.artifact.Files
}
