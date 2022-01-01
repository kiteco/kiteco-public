package pipeline

import (
	"fmt"
)

// PipeClone maintains a partial copy of a Pipeline, in which some Feeds may be clones of their originals.
// It maintains a bidirectional mapping so that the original Feeds can be retrieved given their clones and vice versa.
type PipeClone struct {
	Sources    []Source
	Parents    ParentMap
	Dependents DependentMap

	OrigToClone map[Feed]Feed
	CloneToOrig map[Feed]Feed
}

// CloneForShard returns a pipeline that is applicable to a shard instance.
func (p Pipeline) CloneForShard(shard int, totalShards int) (PipeClone, error) {
	return clonePipe(p.Sources, p.Parents, func(f Feed) (Feed, error) {
		switch f := f.(type) {
		case Source:
			c, err := f.ForShard(shard, totalShards)
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, fmt.Errorf("nil clone returned")
			}
			return c, nil
		case Aggregator:
			c, err := f.ForShard(shard, totalShards)
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, fmt.Errorf("nil clone returned")
			}
			return c, nil
		default:
			return f, nil
		}
	})
}

// CloneForWorker returns a pipieline that is applicable to a specific worker.
func (p PipeClone) CloneForWorker() (PipeClone, error) {
	return clonePipe(p.Sources, p.Parents, func(f Feed) (Feed, error) {
		switch f := f.(type) {
		case Dependent:
			clone := f.Clone()
			if clone == nil {
				return nil, fmt.Errorf("nil clone returned")
			}
			if _, ok := f.(Transform); ok {
				trans, ok := clone.(Transform)
				if !ok {
					return nil, fmt.Errorf("clone of Transform is not a Transform")
				}
				if trans == nil {
					return nil, fmt.Errorf("clone of Transform is a nil Transform")
				}
			}
			if _, ok := f.(Aggregator); ok {
				agg, ok := clone.(Aggregator)
				if !ok {
					return nil, fmt.Errorf("clone of Aggregator is not a Aggregator")
				}
				if agg == nil {
					return nil, fmt.Errorf("clone of Aggregator is a nil Aggregator")
				}
			}
			return clone, nil
		default:
			return f, nil
		}
	})
}

// Aggregators returns all the (possibly cloned) aggregators in the clone
func (p PipeClone) Aggregators() []Aggregator {
	var aggs []Aggregator

	for feed := range p.Parents {
		if s, ok := feed.(Aggregator); ok {
			aggs = append(aggs, s)
		}
	}

	return aggs
}

func clonePipe(sources []Source, parents ParentMap, cloneFn func(Feed) (Feed, error)) (PipeClone, error) {
	origToClone := make(map[Feed]Feed)
	cloneToOrig := make(map[Feed]Feed)

	newSources := make([]Source, 0, len(sources))

	for _, source := range sources {
		c, err := cloneFn(source)
		if err != nil {
			return PipeClone{}, fmt.Errorf("could not clone source %v: %v", source, err)
		}
		newSrc := c.(Source)

		newSources = append(newSources, newSrc)
		origToClone[source] = newSrc
		cloneToOrig[newSrc] = source
	}

	for dep := range parents {
		c, err := cloneFn(dep)
		if err != nil {
			return PipeClone{}, fmt.Errorf("could not clone dependent %v: %v", dep, err)
		}
		newDep := c.(Dependent)

		origToClone[dep] = newDep
		cloneToOrig[newDep] = dep
	}

	newParents := make(ParentMap, len(parents))

	for dep, parent := range parents {
		newFeed, found := origToClone[dep]
		if !found {
			newFeed = dep
		}
		newDep := newFeed.(Dependent)

		newParent, found := origToClone[parent]
		if !found {
			newParent = parent
		}
		newParents[newDep] = newParent
	}

	newDependents := NewDependentMap(newParents)

	return PipeClone{
		Sources:     newSources,
		Parents:     newParents,
		Dependents:  newDependents,
		OrigToClone: origToClone,
		CloneToOrig: cloneToOrig,
	}, nil
}
