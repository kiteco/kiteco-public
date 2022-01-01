package pythonenv

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
)

// FlatItem is a name/type pair
type FlatItem struct {
	Name    string
	ValueID pythontype.FlatID
}

// FlatSourceTree is the representation of source trees used for serialization
type FlatSourceTree struct {
	Files  []FlatItem
	Dirs   []FlatItem
	Values []*pythontype.FlatValue
}

// Inflate creatas a SourceTree from its flattened representation
func (f FlatSourceTree) Inflate(graph pythonresource.Manager) (*SourceTree, error) {
	valuesByID, err := pythontype.InflateValues(f.Values, graph)
	if err != nil {
		return nil, err
	}
	tree := SourceTree{
		Files: make(map[string]*pythontype.SourceModule),
		Dirs:  make(map[string]*pythontype.SourcePackage),
	}
	for _, item := range f.Files {
		file, ok := valuesByID[item.ValueID]
		if !ok {
			return nil, fmt.Errorf("no value found for %s (ID was %d)", item.Name, item.ValueID)
		}
		tree.Files[item.Name], ok = file.(*pythontype.SourceModule)
		if !ok {
			return nil, fmt.Errorf("value for %s was %T not *SourceModule", item.Name, file)
		}
	}
	for _, item := range f.Dirs {
		dir, ok := valuesByID[item.ValueID]
		if !ok {
			return nil, fmt.Errorf("no value found for %s (ID was %d)", item.Name, item.ValueID)
		}
		tree.Dirs[item.Name], ok = dir.(*pythontype.SourcePackage)
		if !ok {
			return nil, fmt.Errorf("value for %s was %T not *SourcePackage", item.Name, dir)
		}
	}
	return &tree, nil
}
