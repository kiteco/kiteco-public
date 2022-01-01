package jsonutil

// jsonutil is now only for backwards compatibility - use serialization.Decode(path, handler) instead.

import "github.com/kiteco/kiteco/kite-golib/serialization"

// ErrStop is a special value returned from handlers to cease processing
var ErrStop = serialization.ErrStop

// DecodeAllFrom loads a series of JSON objects from a file. If the path ends with ".gz" then
// the contents will be automatically decompressed. This function accepts both S3 and local
// paths.
//
// Example:
//
//   var numbers []int
//   err := DecodeAllFrom("/tmp/numbers.json.gz", func(num *Number) error {
//     numbers = append(numbers, num.X)
//     return nil
//   })
func DecodeAllFrom(path string, handler interface{}) error {
	return serialization.Decode(path, handler)
}
