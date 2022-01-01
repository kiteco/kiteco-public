package data

import (
	"testing"

	"github.com/minio/highwayhash"
	"github.com/stretchr/testify/require"
)

func Test_BufferHashNoCR(t *testing.T) {
	text := "line 0 \n line 1 \n line 2 \n unterminated"
	hash, _ := highwayhash.New128(hashKey[:])
	hash.Write([]byte(text))
	require.Equal(t, newBufferHash(hash), NewBuffer(text).Hash())
}

func Test_BufferHashCRNotEqualAtEOF(t *testing.T) {
	crlf := "line 0 \r\n line 1 \r\n line 2 \r\n unterminated"
	lf := "line 0 \n line 1 \n line 2 \n different"
	require.NotEqual(t, NewBuffer(lf).Hash(), NewBuffer(crlf).Hash())
}

func Test_BufferHashCRNotEqual(t *testing.T) {
	crlf := "line 0 \r\n line 1 \r\n line 2 \r\n unterminated"
	lf := "line 0 \n line 1 \n line 2 blah \n unterminated"
	require.NotEqual(t, NewBuffer(lf).Hash(), NewBuffer(crlf).Hash())
}

func Test_BufferHashCREqual(t *testing.T) {
	crlf := "line 0 \r\n line 1 \r\n line 2 \r\n unterminated"
	lf := "line 0 \n line 1 \n line 2 \n unterminated"
	require.Equal(t, NewBuffer(lf).Hash(), NewBuffer(crlf).Hash())
}
