package diskmapindex

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// Index wraps a set of backing diskmaps
type Index struct {
	blocks      []*diskmap.Map
	keyToBlocks map[string][]int
}

// NewIndex from the specified directory
func NewIndex(path, cacheDir string) (*Index, error) {
	entries, err := fileutil.ListDir(path)
	if err != nil {
		return nil, fmt.Errorf("error listing entries in %s: %v", path, err)
	}

	var metaFileName string
	for _, entry := range entries {
		if strings.HasSuffix(entry, metaFileSuffix) {
			if metaFileName != "" {
				return nil, fmt.Errorf("dir %s contains multiple meta files: %s and %s", path, metaFileName, entry)
			}
			metaFileName = entry
		}
	}

	if metaFileName == "" {
		return nil, fmt.Errorf("unable to find meta file in %s", path)
	}

	f, err := readCloser(metaFileName, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", metaFileName, err)
	}
	defer f.Close()

	var meta metaInfo
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		return nil, fmt.Errorf("error decoding json from %s: %v", metaFileName, err)
	}

	for i, b := range meta.Blocks {
		meta.Blocks[i] = fileutil.Join(path, b)
	}

	diskMaps, err := decodeDiskMaps(meta.Blocks, cacheDir, meta.Compress)
	if err != nil {
		return nil, fmt.Errorf("error decoding diskmaps in %s: %v", path, err)
	}

	return &Index{
		blocks:      diskMaps,
		keyToBlocks: meta.KeyToBlocks,
	}, nil
}

// Get the entries for the specified key
func (i *Index) Get(k string) ([][]byte, error) {
	blocks := i.blocks
	if bs, ok := i.keyToBlocks[k]; ok {
		blocks = make([]*diskmap.Map, 0, len(bs))
		for _, b := range bs {
			blocks = append(blocks, i.blocks[b])
		}
	}

	var bufs [][]byte
	for _, blk := range blocks {
		buf, err := blk.Get(k)
		switch {
		case err == diskmap.ErrNotFound:
			continue
		case err != nil:
			return nil, fmt.Errorf("unhandled error in diskmap: %v", err)
		}
		bufs = append(bufs, buf)
	}

	if len(bufs) == 0 {
		return nil, fmt.Errorf("unable to find entry for %s", k)
	}
	return bufs, nil
}

// Keys for all entries in the index, n is used to presize the
// map of keys if an estimated number of keys is known.
func (i *Index) Keys(n int) ([]string, error) {
	var keys []string
	if len(i.keyToBlocks) > 0 {
		keys = make([]string, 0, len(i.keyToBlocks))
		for k := range i.keyToBlocks {
			keys = append(keys, k)
		}
	} else {
		seen := make(map[string]struct{}, n)
		for ii, blk := range i.blocks {
			ks, err := blk.Keys()
			if err != nil {
				return nil, fmt.Errorf("error getting keys for block %d: %v", ii, err)
			}

			for _, k := range ks {
				seen[k] = struct{}{}
			}
		}

		keys = make([]string, 0, len(seen))
		for k := range seen {
			keys = append(keys, k)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

// IterateSlowly scans over the entire diskmap and emits key/value pairs.
// The same key may be emitted multiple times if it appears in multiple blocks.
func (i *Index) IterateSlowly(emit func(key string, val []byte) error) error {
	for _, blk := range i.blocks {
		if err := blk.IterateSlowly(emit); err != nil {
			return err
		}
	}
	return nil
}
