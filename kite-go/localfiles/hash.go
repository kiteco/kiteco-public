package localfiles

import (
	"encoding/base64"
	"hash/fnv"

	"github.com/kiteco/kiteco/kite-golib/bufutil"
)

// Hash represents a hash of a content blob
type Hash uint64

// String returns the hash representation that is sent over the wire and
// stored in the backend database.
func (h Hash) String() string {
	b := bufutil.UintToBytes(uint64(h))
	return base64.URLEncoding.EncodeToString(b)
}

// ParseHash converts the string representation (which is sent over the wire
// and stored in the backend DB) of a hash to an integer (which is used
// internally by the client-side syncer).
func ParseHash(s string) (Hash, error) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return Hash(0), err
	}
	return Hash(bufutil.BytesToUint(b)), nil
}

// RawHash returns an 64 bit integer representation of the hash
// of a buffer. This can be turned into the hash that is sent over the
// wire and stored in the database by calling the String() member.
func RawHash(content []byte) Hash {
	h := fnv.New64()
	h.Write(content)
	return Hash(h.Sum64())
}

// ComputeHash hashes the contents of a file, converts to bytes,
// and b64 encodes it.
func ComputeHash(content []byte) string {
	return RawHash(content).String()
}
