package pythontype

import (
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/pkg/errors"
)

// ErrNotFound is the error returned by Value.Attr when an attribute is not found
var ErrNotFound = errors.New("attribute was not found")

// ErrTooManySteps is the error returned by Value.Attr when an attribute lookup
// exceeds the maximum number of recursive attribute calls.
var ErrTooManySteps = errors.New("attribute was not found")

// ValueNamespace represents a value and a namespace from which it was retrieved.
type ValueNamespace struct {
	Value     Value
	Namespace Value
}

// AttrResult is the result of calling Value.Attr. There are three cases:
//  - attribute not found: all members are nil
//  - one attribute was found: Single is populated, Multiple is nil
//  - the attribute was resolved within a Union: Single is the zero value,
//    Multiple is populated with the result
// The reason for having both single and multiple versions is to avoid
// allocating a slice in the common case that the value was resolved
// within a single namespace.
type AttrResult struct {
	Single   ValueNamespace
	Multiple []ValueNamespace
}

// Found returns true if the attribute was resolved in at least one namespace. Note
// that even when an attribute was resolved, it may have been resolved to an unknown
// value, so ar.Value could still be nil.
func (ar AttrResult) Found() bool {
	return ar.Single.Namespace != nil || len(ar.Multiple) > 0
}

// ExactlyOne returns true if the attribute was resolved to exactly one value. It returns
// false if the attribute was not resolved at all.
func (ar AttrResult) ExactlyOne() bool {
	return ar.Single.Namespace != nil
}

// Value gets the value of the attribute. If the attribute was resolved in multiple
// namespaces then each of the respective values will be combined as a Union.
func (ar AttrResult) Value() Value {
	if ar.Single.Value != nil {
		return ar.Single.Value
	}
	if len(ar.Multiple) == 0 {
		return nil
	}
	var vals []Value
	for _, vn := range ar.Multiple {
		vals = append(vals, vn.Value)
	}
	// TODO(naman) we never use the Namespace field, so we should just do the Unite as part of Attr when creating the AttrResult
	return Unite(kitectx.TODO(), vals...)
}

// SingleResult creates an AttrResult corresponding to a single result
func SingleResult(value, namespace Value) AttrResult {
	return AttrResult{Single: ValueNamespace{value, namespace}}
}

// UnionResult creates an AttrResult corresponding to multiple values
func UnionResult(pairs []ValueNamespace) AttrResult {
	switch len(pairs) {
	case 0:
		return AttrResult{}
	case 1:
		return SingleResult(pairs[0].Value, pairs[0].Namespace)
	default:
		return AttrResult{Multiple: pairs}
	}
}

// resolveAttr resolves an attribute given an object, its dictionary, and the next objects
// to search (typically the type or base classes of the current object)
func resolveAttr(
	ctx kitectx.CallContext,
	name string,
	self Value,
	dict map[string]Value,
	bases ...Value) (AttrResult, error) {
	ctx.CheckAbort()
	if child, ok := dict[name]; ok {
		return SingleResult(child, self), nil
	}
	for _, v := range bases {
		if v == nil {
			continue
		}
		if res, _ := attr(ctx, v, name); res.Found() {
			return res, nil
		}
	}
	return AttrResult{}, ErrNotFound
}
