package builder

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// Options allow specification of where/how to write resource data and the manifest file
type Options struct {
	ResourceRoot      string
	ResourceExtension string
	ManifestPath      string
}

// DefaultResourceRoot is the default location where resource files get written
const DefaultResourceRoot = "s3://kite-resource-manager/python-v2"

// DefaultOptions provides a sensible default Options for use with NewBuilder
var DefaultOptions = Options{
	ResourceRoot:      DefaultResourceRoot,
	ResourceExtension: "blob",
	ManifestPath:      "manifest.json",
}

// Builder allows for writing resources & building manifests
type Builder struct {
	manifest manifest.Manifest
	opts     Options
}

// New creates a Builder to write to the provided manifest file path
func New(opts Options) Builder {
	if !strings.HasPrefix(opts.ResourceRoot, "s3://") {
		abs, err := filepath.Abs(opts.ResourceRoot)
		if err != nil {
			panic("could not compute absolute path")
		}
		opts.ResourceRoot = abs
	}
	return Builder{
		manifest: make(manifest.Manifest),
		opts:     opts,
	}
}

func (b Builder) getWriter(dist keytypes.Distribution, rName string) (fileutil.NamedWriteCloser, error) {
	fname := time.Now().UTC().Format("2006-01-02T15-04-05") + "." + b.opts.ResourceExtension
	version := dist.Version
	if version == "" {
		version = "_"
	}

	path := fileutil.Join(b.opts.ResourceRoot, dist.Name, version, rName, fname)
	w, err := fileutil.NewBufferedWriter(path)
	if err != nil {
		return nil, err
	}

	return w, nil
}

var nonAlphaNumeric = regexp.MustCompile("[^a-zA-Z0-9]+")

func resourceName(r resources.Resource) string {
	// take the final component of the type's package path and the type name
	t := reflect.TypeOf(r)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	parts := strings.Split(t.PkgPath(), "/")
	tentative := fmt.Sprintf("%s_%s", parts[len(parts)-1], t.Name())
	// remove all non-alphanumeric characters
	return nonAlphaNumeric.ReplaceAllString(tentative, "")
}

// PutResource is used by resource builders/producers to write a resource to disk/s3,
// and update the in-memory manifest to point at the newly written resource
func (b Builder) PutResource(dist keytypes.Distribution, r resources.Resource) error {
	rName := resourceName(r)
	w, err := b.getWriter(dist, rName)
	if err != nil {
		return err
	}
	err = r.Encode(w)
	if err != nil {
		w.Close()
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	locator, exists := b.manifest[dist]
	if !exists {
		locator = make(resources.LocatorGroup)
		b.manifest[dist] = locator
	}

	err = locator.Update(r, resources.Locator(w.Name()))
	if err != nil {
		return err
	}

	return nil
}

// Commit writes the in-memory manifest to disk and uses go-bindata to regenerate the Asset file
func (b Builder) Commit() error {
	w, err := fileutil.NewBufferedWriter(b.opts.ManifestPath)
	if err != nil {
		return err
	}
	defer w.Close()

	err = b.manifest.Encode(w)
	if err != nil {
		return err
	}

	return nil
}
