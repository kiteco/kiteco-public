package datadeps

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"io"
	"log"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// AssetFileMap implements FileMap using Assets
type AssetFileMap struct {
	offsets map[string]fileutil.Offset
}

// NewAssetFileMap creates an AssetFileMap
func NewAssetFileMap() (fileutil.FileMap, error) {
	af := &AssetFileMap{}
	data, err := Offsets()
	if err != nil {
		return nil, err
	}

	var fileOffsets []fileutil.FileOffset
	b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(data))
	decoder := gob.NewDecoder(b64)
	if err := decoder.Decode(&fileOffsets); err != nil {
		return nil, err
	}
	af.offsets = make(map[string]fileutil.Offset)
	for _, fo := range fileOffsets {
		af.offsets[fo.Path] = fo.Offset
	}
	return af, nil
}

var assetFileMap fileutil.FileMap

// UseAssetFileMap caches the decompressed AssetFileMap in memory, and configures fileutil to use it; it is idempotent
func UseAssetFileMap() error {
	var err error
	if assetFileMap == nil {
		assetFileMap, err = NewAssetFileMap()
		if err != nil {
			return err
		}
	}

	fileutil.SetLocalFileMap(assetFileMap)

	return nil
}

// Enable is a memorable alias for UseAssetFileMap
var Enable = UseAssetFileMap

// SetLocalOnly aliases fileutil.SetLocalOnly
var SetLocalOnly = fileutil.SetLocalOnly

// GetOffset returns the offset for the given path, implements fileutil.FileMap
func (af *AssetFileMap) GetOffset(path string) (fileutil.Offset, bool) {
	offset, ok := af.offsets[path]
	return offset, ok
}

// GetDataFile returns the data file asset, implements fileutil.FileMap
func (af *AssetFileMap) GetDataFile() (io.ReadSeeker, error) {
	data, err := Datadeps()
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// --

// Hash computes a 64-bit hash from two data sources (filemap and offsets)
func Hash(data1, data2 []byte) uint64 {
	return spooky.Hash64Seed(data1, spooky.Hash64(data2))
}

// CurrentHash computes the hash of the current dataset
func CurrentHash() (uint64, error) {
	data1, err := Datadeps()
	if err != nil {
		return 0, err
	}

	log.Printf("current datadeps: %x", spooky.Hash64(data1))

	data2, err := Offsets()
	if err != nil {
		return 0, err
	}

	log.Printf("current offsets: %x", spooky.Hash64(data2))

	return Hash(data1, data2), nil
}
