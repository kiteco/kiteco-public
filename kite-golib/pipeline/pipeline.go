package pipeline

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
)

// Pipeline defines a pipeline that can be run via the engine.
type Pipeline struct {
	Name string
	// Parents maps each non-Source feed to its parent. Note that Sources cannot have parents.
	Parents ParentMap
	// Sources contains the sources from which records will be generated.
	Sources []Source
	// Params represents the parameters used to create the pipeline.
	Params map[string]interface{}
	// ResultsFn is an optional function that takes in the final aggregated value of each Aggregator and returns
	// a slice of Results to be recorded for the pipeline run.
	ResultsFn func(res map[Aggregator]Sample) []rundb.Result
}

// ParentMap represents a map of dependent feeds to their parents from which they take their data.
// TODO: easy way to relax this assumption?
type ParentMap map[Dependent]Feed

// Chain feeds to one another, such that a feed uses the data from its predecessor. The last feed given is returned.
func (p ParentMap) Chain(emitter Feed, firstDep Dependent, otherDeps ...Dependent) Dependent {
	deps := []Dependent{firstDep}
	deps = append(deps, otherDeps...)

	for i, dep := range deps {
		parent := emitter
		if i > 0 {
			parent = deps[i-1]
		}
		p[dep] = parent
	}

	return deps[len(deps)-1]
}

// FanOut broadcasts samples from the emitter to all the other provided dependents
func (p ParentMap) FanOut(emitter Feed, firstDep Dependent, otherDeps ...Dependent) {
	deps := []Dependent{firstDep}
	deps = append(deps, otherDeps...)
	for _, dep := range deps {
		p[dep] = emitter
	}
}

// Validate that the pipeline is correct.
func (p Pipeline) Validate() error {
	if len(p.Name) == 0 {
		return fmt.Errorf("pipeline name cannot be empty")
	}

	for i, s := range p.Sources {
		if err := validateFeed(s); err != nil {
			return fmt.Errorf("invalid source %d %v: %v", i, s, err)
		}
	}

	for dep, parent := range p.Parents {
		if err := validateFeed(dep); err != nil {
			return fmt.Errorf("invalid dependent %v with parent %v: %v", dep, parent, err)
		}

		if err := validateFeed(parent); err != nil {
			return fmt.Errorf("invalid parent %v with dependent %v: %v", parent, dep, err)
		}
	}

	// check uniqueness of names
	names := make(map[string]struct{})

	for _, feed := range p.AllFeeds() {
		if _, found := names[feed.Name()]; found {
			return fmt.Errorf("duplicate name for feed: %s", feed.Name())
		}
		names[feed.Name()] = struct{}{}
	}

	return nil
}

// Aggregators returns all aggregators in the pipeline.
func (p Pipeline) Aggregators() []Aggregator {
	var aggs []Aggregator

	for feed := range p.Parents {
		if s, ok := feed.(Aggregator); ok {
			aggs = append(aggs, s)
		}
	}

	sort.Slice(aggs, func(i, j int) bool {
		return aggs[i].Name() < aggs[j].Name()
	})

	return aggs
}

// AllFeeds used in the pipeline
func (p Pipeline) AllFeeds() []Feed {
	var feeds []Feed

	for _, source := range p.Sources {
		feeds = append(feeds, source)
	}

	for dep := range p.Parents {
		feeds = append(feeds, dep)
	}

	sort.Slice(feeds, func(i, j int) bool {
		return feeds[i].Name() < feeds[j].Name()
	})

	return feeds
}

// DependentMap maps each feed to its dependents.
type DependentMap map[Feed][]Dependent

// NewDependentMap given a map of each feed to its parent
func NewDependentMap(pm ParentMap) DependentMap {
	deps := make(DependentMap)

	for dep, parent := range pm {
		deps[parent] = append(deps[parent], dep)
	}

	for _, d := range deps {
		sort.Slice(d, func(i, j int) bool {
			return d[i].Name() < d[j].Name()
		})
	}

	return deps
}

func validateFeed(f Feed) error {
	if f.Name() == "" {
		return fmt.Errorf("feed %v must have a non-empty name", f)
	}

	if reflect.ValueOf(f).Kind() != reflect.Ptr {
		return fmt.Errorf("feed %v is not a pointer", f)
	}

	if reflect.ValueOf(f).IsNil() {
		return fmt.Errorf("feed is nil")
	}

	switch f := f.(type) {
	case Source:
	case Dependent:
	default:
		return fmt.Errorf("feed %v needs to be a Source or Dependent", f.Name())
	}

	return nil
}
