package resources

import (
	"reflect"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/pkg/errors"
)

// verify that the Group type is well-formed
func init() {
	rgTy := reflect.TypeOf((*Group)(nil)).Elem()
	for i := 0; i < rgTy.NumField(); i++ {
		if !rgTy.Field(i).Type.Implements(reflect.TypeOf((*Resource)(nil)).Elem()) {
			panic("resources Group struct member has non-Resource type")
		}
	}
}

// Locator supports loading a resource
type Locator string

// Load populates the provided resource
func (l Locator) Load(rs Resource) error {
	if l == "" {
		return nil // nothing to load
	}

	r, err := fileutil.NewCachedReader(string(l))
	if err != nil {
		return err
	}
	defer r.Close()

	return rs.Decode(r)
}

// LocatorGroup encapsulates loading of a Group
type LocatorGroup map[string]Locator

// Load opens and decodes all available resources from s3/disk
func (lg LocatorGroup) Load(rg Group) error {
	// using reflection here shouldn't be a problem as the performance bottleneck is surely I/O
	rgVal := reflect.ValueOf(rg)
	for key, l := range lg {
		fieldVal := rgVal.FieldByName(key)
		if !fieldVal.IsValid() {
			return errors.Errorf("field %s not found in resources Group struct", key)
		}
		if err := l.Load(fieldVal.Interface().(Resource)); err != nil {
			return errors.Wrapf(err, "failed to load field %s on resources Group", key)
		}
	}
	return nil
}

// Update updates the Group locator for a given resource and resource locator
// for use by pythonresource/build
func (lg LocatorGroup) Update(r Resource, l Locator) error {
	if lg == nil {
		// ok to panic, since this codepath should only run offline
		panic("cannot update a nil LocatorGroup")
	}
	rTy := reflect.TypeOf(r)
	rgTy := reflect.TypeOf((*Group)(nil)).Elem()
	for i := 0; i < rgTy.NumField(); i++ {
		f := rgTy.Field(i)
		if f.Type == reflect.TypeOf(r) {
			lg[f.Name] = l
			return nil
		}
	}

	return errors.Errorf("resource type %s not present in resources Group struct", rTy)
}
