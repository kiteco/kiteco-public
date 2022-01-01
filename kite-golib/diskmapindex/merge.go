package diskmapindex

import (
	"fmt"
)

// MergeFn merges a list of values for the same
// key and returns a new value
type MergeFn func(values [][]byte) ([]byte, error)

// MergeOptions ...
type MergeOptions struct {
	Builder            BuilderOptions
	CacheDir           string
	MaxBlockSizeBytes  int
	WaitOnBlockWriting bool
}

// Merge constructs a new index by taking as input an index,
// getting all the values for each key in the index and then
// writing these values to a new block
func Merge(opts MergeOptions, in, out string, merge MergeFn) error {
	idx, err := NewIndex(in, opts.CacheDir)
	if err != nil {
		return fmt.Errorf("error opening input index %s: %v", in, err)
	}

	keys, err := idx.Keys(0)
	if err != nil {
		return err
	}

	builder := NewBuilder(opts.Builder, out)

	var kvs []KeyValue
	var size int
	flush := func() {
		l := len(kvs)
		builder.AddBlock(kvs, opts.WaitOnBlockWriting)
		kvs = make([]KeyValue, 0, l)
		size = 0
	}

	for _, k := range keys {
		vals, err := idx.Get(k)
		if err != nil {
			return err
		}

		val, err := merge(vals)
		if err != nil {
			return fmt.Errorf("error merging values for %s: %v", k, err)
		}

		if opts.MaxBlockSizeBytes > 0 && len(val)+size > opts.MaxBlockSizeBytes {
			flush()
		}

		kvs = append(kvs, KeyValue{
			Key:   k,
			Value: val,
		})
		size += len(val)
	}

	flush()

	builder.Finalize()

	if err := builder.Err(); err != nil {
		return fmt.Errorf("builder encountered error(s): %v", err)
	}

	return nil
}
