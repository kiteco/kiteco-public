package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

type codeStore struct {
	hashToCode      *pythoncode.HashToSourceIndex
	symbToHash      *pythoncode.SymbolToHashesIndex
	canonSymbToHash *pythoncode.SymbolToHashesIndex
}

func newCodeStore(cache string) (*codeStore, error) {
	hashToCode, err := pythoncode.NewHashToSourceIndex(pythoncode.HashToSourceIndexPath, cache)
	if err != nil {
		return nil, fmt.Errorf("error loading hash to source index from %s: %v", pythoncode.HashToSourceIndexPath, err)
	}

	symbToHash, err := pythoncode.NewSymbolToHashesIndex(pythoncode.SymbolToHashesIndexPath, cache)
	if err != nil {
		return nil, fmt.Errorf("error loading symbol to hashes index from %s: %v", pythoncode.SymbolToHashesIndexPath, err)
	}

	canonSymbToHash, err := pythoncode.NewSymbolToHashesIndex(pythoncode.CanonicalSymbolToHashesIndexPath, cache)
	if err != nil {
		return nil, fmt.Errorf("error loading canonical symbol to hashes index from %s: %v", pythoncode.CanonicalSymbolToHashesIndexPath, err)
	}

	return &codeStore{
		hashToCode:      hashToCode,
		symbToHash:      symbToHash,
		canonSymbToHash: canonSymbToHash,
	}, nil
}

func (c *codeStore) HashesFor(sym pythonresource.Symbol, useCanonical, sortHashes bool) ([]pythoncode.HashCounts, error) {
	defer fetchHashDuration.DeferRecord(time.Now())

	if useCanonical {
		sym = sym.Canonical()
	}

	idx := c.symbToHash
	if useCanonical {
		idx = c.canonSymbToHash
	}

	hashes, err := idx.HashesFor(sym)
	if err != nil {
		return nil, err
	}

	// sort the hashes alphabetically
	if sortHashes {
		sort.Slice(hashes, func(i, j int) bool {
			return hashes[i].Hash < hashes[j].Hash
		})
	}

	return hashes, nil
}

func (c *codeStore) SourceFor(hash string) ([]byte, error) {
	defer fetchSourceDuration.DeferRecord(time.Now())

	src, err := c.hashToCode.SourceFor(hash)
	if err != nil {
		return nil, err
	}

	return src, nil
}
