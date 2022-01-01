package pipeline

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustGetEngine(t *testing.T, parents ParentMap, sources ...Source) *Engine {
	return mustGetEngineWithOpts(t, DefaultEngineOptions, parents, sources...)
}

func mustGetEngineWithOpts(t *testing.T, opts EngineOptions, parents ParentMap, sources ...Source) *Engine {
	pipe := Pipeline{Name: "test", Parents: parents, Sources: sources}
	opts.NoServer = true
	e, err := NewEngine(pipe, opts)
	require.NoError(t, err)
	return e
}

type intSample int

func (i intSample) SampleTag() {}

type intList struct {
	l   []int
	pos int
}

// intList implements Source
func (i *intList) SourceOut() Record {
	if i.pos >= len(i.l) {
		return Record{}
	}

	s := i.l[i.pos]
	i.pos++
	rec := Record{
		Key:   strconv.Itoa(i.pos),
		Value: intSample(s),
	}
	return rec
}

// intList implements Source
func (i *intList) Name() string {
	return "intList"
}

// intList implements Source
func (i *intList) ForShard(shard, totalShards int) (Source, error) {
	if totalShards != 1 {
		panic("sharding not implemented")
	}

	return &intList{
		l: i.l,
	}, nil
}

type intSum struct {
	s    int
	name string
}

func newIntSum(name string) *intSum {
	return &intSum{
		name: name,
	}
}

// intSum implements Aggregator
func (i *intSum) Name() string {
	return i.name
}

// intSum implements Aggregator
func (i *intSum) ForShard(shard, totalShards int) (Aggregator, error) {
	return &intSum{
		name: i.name,
	}, nil
}

// intSum implements Aggregator
func (i *intSum) Clone() Dependent {
	return &intSum{}
}

// intSum implements Aggregator
func (i *intSum) In(s Sample) {
	i.s += int(s.(intSample))
}

// intSum implements Aggregator
func (i *intSum) AggregateLocal(clones []Aggregator) (Sample, error) {
	var sum int

	for _, c := range clones {
		sum += c.(*intSum).s
	}

	return intSample(sum), nil
}

// intSum implements Aggregator
func (intSum) FromJSON([]byte) (Sample, error) {
	panic("not implemented")
}

// intSum implements Aggregator
func (intSum) AggregateFromShard(Sample, Sample, string) (Sample, error) {
	panic("not implemented")
}

// Finalize implements Aggregator
func (intSum) Finalize() error {
	return nil
}

type repeater struct {
	n int

	s        Sample
	repeated int
}

// repeater implements Transform
func (r *repeater) Name() string {
	return "repeater"
}

// repeater implements Transform
func (r *repeater) In(s Sample) {
	r.s = s
	r.repeated = 0
}

// repeater implements Transform
func (r *repeater) TransformOut() Sample {
	if r.repeated == r.n {
		return nil
	}

	r.repeated++
	return r.s
}

// repeater implements Transform
func (r *repeater) Clone() Dependent {
	return &repeater{n: r.n}
}

func intResult(s Sample) int {
	return int(s.(intSample))
}

func TestPipelineTwoStage(t *testing.T) {
	source := &intList{
		l: []int{1, 2, 3},
	}

	agg := newIntSum("agg")

	e := mustGetEngine(t, map[Dependent]Feed{
		agg: source,
	}, source)

	res, err := e.Run()
	assert.Nil(t, err)

	assert.Equal(t, 6, intResult(res[agg]))
}

func TestPipelineThreeStage(t *testing.T) {
	source := &intList{
		l: []int{1, 2, 3},
	}

	r := &repeater{
		n: 2,
	}

	agg := newIntSum("agg")

	e := mustGetEngine(t, map[Dependent]Feed{
		r:   source,
		agg: r,
	}, source)

	res, err := e.Run()
	assert.Nil(t, err)

	assert.Equal(t, 12, intResult(res[agg]))
}

func TestPipelineSplit(t *testing.T) {
	source := &intList{
		l: []int{1, 2, 3},
	}

	r := &repeater{
		n: 2,
	}

	agg1 := newIntSum("agg1")
	agg2 := newIntSum("agg2")

	e := mustGetEngine(t, map[Dependent]Feed{
		r:    source,
		agg1: r,
		agg2: r,
	}, source)

	res, err := e.Run()
	assert.Nil(t, err)

	assert.Equal(t, 12, intResult(res[agg1]))
	assert.Equal(t, 12, intResult(res[agg2]))
}

func TestOnlyKeys(t *testing.T) {
	source := &intList{
		l: []int{1, 2, 3},
	}
	r := &repeater{
		n: 2,
	}
	agg := newIntSum("agg")
	pm := make(ParentMap)
	pm.Chain(source, r, agg)

	opts := DefaultEngineOptions
	opts.OnlyKeys = map[string][]string{source.Name(): {"2"}}
	e := mustGetEngineWithOpts(t, opts, map[Dependent]Feed{
		r:   source,
		agg: r,
	}, source)

	res, err := e.Run()
	assert.Nil(t, err)

	assert.Equal(t, 4, intResult(res[agg]))
}
