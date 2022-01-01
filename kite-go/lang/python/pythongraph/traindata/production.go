package traindata

import (
	"fmt"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

const (
	// ChooseKwargPrefix is a prefix used for all choose kwarg keys
	ChooseKwargPrefix = "choose_kwarg"

	// ChooseArgTypePrefix is a prefix used for all choose argtype keys
	ChooseArgTypePrefix = "choose_argtype"

	// ChooseArgPlaceholderPrefix is a prefix for all choose argplaceholder keys
	ChooseArgPlaceholderPrefix = "choose_arg_placeholder"
)

func hash(s string) pythonimports.Hash {
	return pythonimports.PathHash([]byte(s))
}

// IDForChooseArgTypeParent returns the id for a "choose arg type" parent node
func IDForChooseArgTypeParent(sym string) pythonimports.Hash {
	return hash(fmt.Sprintf("%s:%s", ChooseArgTypePrefix, sym))
}

// IDForChooseArgType returns the id for a "choose arg type" node
func IDForChooseArgType(sym string, at ArgType) pythonimports.Hash {
	return hash(fmt.Sprintf("%s:%s:%s", ChooseArgTypePrefix, sym, at))
}

// IDForChooseArgPlaceholderParent returns the id string for a choose arg placeholder parent node
// in a choose placeholder task
func IDForChooseArgPlaceholderParent(sym string) pythonimports.Hash {
	return hash(fmt.Sprintf("%s:%s", ChooseArgPlaceholderPrefix, sym))
}

// IDForChooseArgPlaceholder returns the id string for a choose arg placeholder node
func IDForChooseArgPlaceholder(sym string, argName string, ap ArgPlaceholder) pythonimports.Hash {
	return hash(fmt.Sprintf("%s:%s:%s:%s", ChooseArgPlaceholderPrefix, sym, argName, ap))
}

// IDForChooseKwargParent returns the id for the given func symbol
// in a choose kwarg task
func IDForChooseKwargParent(sym string) pythonimports.Hash {
	return hash(fmt.Sprintf("%s:%s", ChooseKwargPrefix, sym))
}

// IDForChooseKwarg returns the id for a "choose kwarg" node
func IDForChooseKwarg(sym string, kn string) pythonimports.Hash {
	return hash(fmt.Sprintf("%s:%s:%s", ChooseKwargPrefix, sym, kn))
}

// ArgType is the type of an argument, it can be positional/keyword/stop
type ArgType string

const (
	// Stop indicates the argument type stop (no more arguments)
	Stop = ArgType("stop")
	// Positional indicates a positional argument
	Positional = ArgType("positional")
	// Keyword indicates a keyword argument
	Keyword = ArgType("keyword")
)

// ArgTypes is a list of possible ArgType
var ArgTypes = [...]ArgType{Stop, Positional, Keyword}

// ArgPlaceholder is the type associated with the 2 candidates used for the ArgPlaceholder task
type ArgPlaceholder string

const (
	// Placeholder represents the candidate for putting a placeholder for a parameter
	Placeholder = ArgPlaceholder("placeholder")
	// NoPlaceholder represents the candidate for trying to find a name for a parameter
	NoPlaceholder = ArgPlaceholder("no_placeholder")
)

// ArgPlaceholders is the list of the 2 candidates used for the ArgPlaceholder task
var ArgPlaceholders = [...]ArgPlaceholder{Placeholder, NoPlaceholder}

// Production in a "formal grammar".
// Notes:
//   - if len(Children) == 0 then this is a "terminal node"
type Production struct {
	ID pythonimports.Hash `json:"id"`
	// Children contains the IDs of the productions that the parent
	// non terminal can be expanded to
	Children []pythonimports.Hash `json:"children"`
}

// ProductionIndexBuilder handles constructing a production index
type ProductionIndexBuilder struct {
	productions map[pythonimports.Hash]Production
}

// NewProductionIndexBuilder ...
func NewProductionIndexBuilder() ProductionIndexBuilder {
	return ProductionIndexBuilder{
		productions: make(map[pythonimports.Hash]Production),
	}
}

// Add a production
func (pib ProductionIndexBuilder) Add(p Production, ensureUnique bool) error {
	if _, ok := pib.productions[p.ID]; !ok {
		pib.productions[p.ID] = p
	} else if ensureUnique {
		return fmt.Errorf("already saw id %s", p.ID)
	}
	for _, child := range p.Children {
		if cp, ok := pib.productions[child]; ok {
			if ensureUnique {
				return fmt.Errorf("already saw child %s with parent '%s', old prod is %v", child, p.ID, cp)
			}
			continue
		}

		pib.productions[child] = Production{
			ID: child,
		}
	}
	return nil
}

// Finalize the builder and return an index
func (pib ProductionIndexBuilder) Finalize() ProductionIndex {
	keys := make([]pythonimports.Hash, 0, len(pib.productions))
	for id := range pib.productions {
		keys = append(keys, id)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	// TODO: should we have better locality properties for
	// children of the same parent?
	indices := make(map[pythonimports.Hash]int32, len(keys))
	for i, k := range keys {
		indices[k] = int32(i)
		// make sure production children are never nil
		prod := pib.productions[k]
		prod.Children = append([]pythonimports.Hash{}, prod.Children...)
		pib.productions[k] = prod
	}

	return ProductionIndex{
		Indices:     indices,
		Productions: pib.productions,
	}
}

// ProductionIndex manages a set of productions
type ProductionIndex struct {
	Indices map[pythonimports.Hash]int32 `json:"indices"`

	// - training only
	Productions map[pythonimports.Hash]Production `json:"productions"`
}

// ForInference deletes data not required for inference
func (i ProductionIndex) ForInference() ProductionIndex {
	return ProductionIndex{
		Indices: i.Indices,
	}
}

// Production for the particular parent symbol
func (i ProductionIndex) Production(parent pythonimports.Hash) (Production, error) {
	prod, ok := i.Productions[parent]
	if !ok {
		return Production{}, fmt.Errorf("unable to find production with parent %s", parent)
	}
	return prod, nil
}

// Children indices for a particular parent symbol
func (i ProductionIndex) Children(parent pythonimports.Hash) ([]int32, error) {
	prod, err := i.Production(parent)
	if err != nil {
		return nil, err
	}

	children := make([]int32, 0, len(prod.Children))
	for _, child := range prod.Children {
		idx, ok := i.Indices[child]
		if !ok {
			return nil, fmt.Errorf("unable to find index for child %s of parent %s", child, parent)
		}
		children = append(children, idx)
	}
	return children, nil
}

// ChildrenWithLabel returns the indices for the children symbols along with the
// index of the label (child) in the resulting slice.
func (i ProductionIndex) ChildrenWithLabel(parent, child pythonimports.Hash) ([]int32, int, error) {
	children, err := i.Children(parent)
	if err != nil {
		return nil, 0, err
	}

	childIdx, ok := i.Indices[child]
	if !ok {
		return nil, 0, fmt.Errorf("unable to find index for child %s of parent %s", child, parent)
	}

	for i, idx := range children {
		if idx == childIdx {
			return children, i, nil
		}
	}

	return nil, 0, fmt.Errorf("unable to find child %s of parent %s", child, parent)
}

// Index for the specified symbol
func (i ProductionIndex) Index(sym pythonimports.Hash) (int32, bool) {
	idx, ok := i.Indices[sym]
	return idx, ok
}

// MustGetIndices for the provided symbols
func (i ProductionIndex) MustGetIndices(syms ...pythonimports.Hash) []int32 {
	idxs := make([]int32, 0, len(syms))
	for _, s := range syms {
		idx, ok := i.Index(s)
		if !ok {
			panic(fmt.Sprintf("unable to get index for %v", s))
		}
		idxs = append(idxs, idx)
	}
	return idxs
}
