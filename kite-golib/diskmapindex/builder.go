package diskmapindex

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"

	"github.com/kiteco/kiteco/kite-golib/diskmap"
)

const metaFileSuffix = "meta"

// BuilderOptions ...
type BuilderOptions struct {
	Compress         bool
	BuildKeyToBlocks bool
	CacheDir         string
}

type metaInfo struct {
	Compress    bool
	Blocks      []string
	KeyToBlocks map[string][]int
}

// KeyValue is a string key and a []byte value
type KeyValue struct {
	Key   string
	Value []byte
}

// Builder builds an index that is backed by a set of diskmaps
// where each diskmap corresponds to a single in memory map from
// string -> []byte
type Builder struct {
	out  string
	opts BuilderOptions

	blocks      []string
	errs        errors.Errors
	keyToBlocks map[string][]int

	m  sync.Mutex
	wg sync.WaitGroup
}

// NewBuilder using the specified output directory and options
func NewBuilder(opts BuilderOptions, out string) *Builder {
	return &Builder{
		out:         out,
		opts:        opts,
		keyToBlocks: make(map[string][]int),
	}
}

// AddBlock to the index, kvs is NOT copied and
// will be read to and written from in another
// go routine, callers should NOT use kvs after calling this function.
func (b *Builder) AddBlock(kvs []KeyValue, wait bool) {
	idx := b.getNextBlock()

	outName := fmt.Sprintf("block-%04d", idx)
	outPath := fileutil.Join(b.out, outName)

	b.wg.Add(1)
	add := func() {
		defer b.wg.Done()
		fmt.Println("starting to write block to", outPath)

		outf, err := fileutil.NewBufferedWriterWithCache(outPath, b.opts.CacheDir)
		if err != nil {
			b.addErr(fmt.Errorf("error creating file %s: %v", outPath, err))
			return
		}

		out := io.WriteCloser(outf)
		if b.opts.Compress {
			out = gzip.NewWriter(out)
		}

		start := time.Now()

		sort.Slice(kvs, func(i, j int) bool {
			return kvs[i].Key < kvs[j].Key
		})

		sb := diskmap.NewStreamBuilder(out)
		for _, kv := range kvs {
			if b.opts.BuildKeyToBlocks {
				b.keyToBlocks[kv.Key] = append(b.keyToBlocks[kv.Key], idx)
			}
			if err := sb.Add(kv.Key, kv.Value); err != nil {
				b.addErr(fmt.Errorf("error adding entry for %s: %v", kv.Key, err))
				return
			}
		}

		if err := sb.Close(); err != nil {
			b.addErr(fmt.Errorf("error closing stream builder for %s: %v", outPath, err))
			return
		}

		if b.opts.Compress {
			erro := out.Close()
			errf := outf.Close()
			if erro != nil {
				b.addErr(fmt.Errorf("error closing gzip: %v", err))
			}

			if errf != nil {
				b.addErr(fmt.Errorf("error closing file %s: %v", outPath, errf))
			}

			if erro != nil || errf != nil {
				return
			}
		} else {
			if err := out.Close(); err != nil {
				b.addErr(fmt.Errorf("error closing file %s: %v", outPath, err))
				return
			}
		}

		b.addBlock(idx, outName)
		fmt.Printf("took %v to write block %s\n", time.Since(start), outPath)
	}

	if wait {
		add()
	} else {
		go add()
	}
}

func (b *Builder) addErr(err error) {
	b.m.Lock()
	defer b.m.Unlock()
	b.errs = errors.Append(b.errs, err)
}

func (b *Builder) addBlock(i int, path string) {
	b.m.Lock()
	defer b.m.Unlock()

	b.blocks[i] = path
}

func (b *Builder) getNextBlock() int {
	b.m.Lock()
	defer b.m.Unlock()

	idx := len(b.blocks)
	b.blocks = append(b.blocks, "")

	return idx
}

// Err encountered by the builder
func (b *Builder) Err() error {
	return b.errs
}

// Finalize the index and write it to the output directory.
// This should only be called once, from a single go routine.
func (b *Builder) Finalize() {
	// wait for any outstanding writes to finish
	b.wg.Wait()

	outPath := fileutil.Join(b.out, metaFileSuffix)
	outf, err := fileutil.NewBufferedWriter(outPath)
	if err != nil {
		b.addErr(fmt.Errorf("error making output file %s: %v", outPath, err))
		return
	}

	meta := metaInfo{
		Compress:    b.opts.Compress,
		Blocks:      b.blocks,
		KeyToBlocks: b.keyToBlocks,
	}

	if err := json.NewEncoder(outf).Encode(meta); err != nil {
		outf.Close()
		b.addErr(fmt.Errorf("error encoding output to %s: %v", outPath, err))
		return
	}

	if err := outf.Close(); err != nil {
		b.addErr(fmt.Errorf("error closing writer for %s: %v", outPath, err))
	}
}
