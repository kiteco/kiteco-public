package pipeline

// The pipeline composed of a dependency graph of Feeds. A Feed may be one of the following:
// * Source - creates records for analysis. Each record contains a key as well as an associated Sample.
// * Dependent - takes in Samples returned by Sources or other Feeds. Special cases of Dependents are:
//   * Transform - takes in Samples as input and emits Sample(s) for output to other Feeds
//   * Aggregator - takes in Samples for input and whose results are aggregated after the pipeline runs. Once the engine
//     finishes running, it returns one final aggregated Sample for each Aggregator in the pipeline.

// Each Dependent should have exactly one parent defined in the pipeline (see ParentMap), whose output is
// used as input for the feed in question. Sources do not have any parents as they are expected to generate the
// source data.

// Note that none of the methods need to be thread-safe, but for a given Feed, multiple clones of it must be able to
// work concurrently without interference.

// A Feed describes any entity that emits and/or processes incoming data. A data pipeline is composed of dependency
// graph of Feeds.
type Feed interface {
	// Name of the feed.
	Name() string
}

// Source describes a Feed that emits records for analysis.
type Source interface {
	Feed
	// ForShard should return a Source that is applicable to the same dataset, but for a specific shard of the dataset
	// in the range (0, totalShards]. The combined outputs of an arbitrary number of shards should equal the
	// output of a single shard (i.e. shard = 0, maxShards = 1). ForShard will only be called once per pipeline process,
	// and thus the Aggregator returned does not need to work concurrently with another returnee of ForShard.
	ForShard(shard int, totalShards int) (Source, error)
	// SourceOut will be repeatedly called until an empty Record struct is returned.
	SourceOut() Record
}

// Record contains a sample emitted by a Source along with some extra metadata.
type Record struct {
	// An optional string that should uniquely identify the record within the dataset.
	Key string
	// The sample for the given key
	Value Sample
}

// Dependent is used to describe Feeds that take in produced by other feeds.
// Only cloned Dependents (see Clone()) are expected to input any data.
type Dependent interface {
	Feed
	// Clone should create a new Dependent that has the same behavior.
	Clone() Dependent
	In(Sample)
}

// Transform describes a Dependent that transforms samples. For every sample received from the parent, In is called
// once, followed by calls to TransformOut until TransformOut returns nil.
// Only cloned Transforms are expected to input or output any data.
type Transform interface {
	Dependent
	TransformOut() Sample
}

// Aggregator describes a Dependent that aggregates results, both locally (from multiple per-worker clones of itself
// on the same instance) and remotely (from multiple shard instances on a distributed pipeline). This aggregation
// happens after the pipeline has finished running. Once the engine has finished running, a Sample will be returned
// for each aggregator.
type Aggregator interface {
	Dependent
	// ForShard returns an Aggregator with the same behavior, but relevant to the current shard. This will be only
	// called once per pipeline process, and thus the Aggregator returned does not need to work concurrently with
	// another returnee of ForShard.
	ForShard(shard int, totalShards int) (Aggregator, error)
	// AggregateLocal is called on the parent (un-cloned) Aggregator on each instance after the pipeline finishes,
	// with the arguments being its clones. It should return a Sample, which in turn may be serialized to JSON if
	// the pipeline is run in a distributed fashion.
	//
	// If the pipeline is run in standalone model, AggregateLocal is expected to bring the aggregation to its final
	// state; AggregateFromShard will not be called.
	AggregateLocal(clones []Aggregator) (Sample, error)
	// FromJSON deserializes (from JSON) the Sample created by AggregateLocal on a remote instance.
	FromJSON(data []byte) (Sample, error)
	// AggregateFromShard aggregates the deserialized sample (as returned by AggregateLocal) that was pulled from a
	// given shard, together with the coordinator's aggregate (agg). endpoint is the endpoint (host:port) of the given
	// shard, in case some out-of-band operations (e.g. copying of files over from the shard instances) need to occur.
	//
	// This is called once for each shard instance. When AggregateFromShard is called on the first shard, agg is nil
	// since no aggregate has been created yet. Once AggregateFromShard is called on the last shard, it should
	// return the final result.
	//
	// This is only called if the pipeline is in a distributed environment; when running in standalone mode,
	// only AggregateLocal will be called for a given aggregator.
	AggregateFromShard(agg Sample, shardSample Sample, endpoint string) (Sample, error)
	// Finalize is called once aggregation has finished and we have the final result; this can perform any out-of-band
	// operations as needed.
	Finalize() error
}
