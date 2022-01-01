package hash

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"

	spooky "github.com/dgryski/go-spooky"
)

// SpookyHash128String returns a base64-encoded string representation of the hash of x
func SpookyHash128String(x []byte) string {
	var h1, h2 uint64
	spooky.Hash128(x, &h1, &h2)
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, h1)
	binary.Write(&buf, binary.LittleEndian, h2)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
