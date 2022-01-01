package awsutil

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEMRWriterReader(t *testing.T) {
	numRecords := 3

	var b bytes.Buffer
	w := NewEMRWriter(&b)
	for i := 0; i < numRecords; i++ {
		key := fmt.Sprintf("%d", i)
		value := fmt.Sprintf("hello %d", i)
		err := w.Emit(key, []byte(value))
		require.NoError(t, err)
	}
	w.Close()

	var i int
	r := NewEMRReader(&b)
	for {
		key, value, err := r.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%d", i), key, "key mismatch")
		assert.Equal(t, fmt.Sprintf("hello %d", i), string(value), "value mismatch")
		i++
	}

	assert.Equal(t, i, numRecords, "record count mismatch")
}

func TestEMRIterator(t *testing.T) {
	numRecords := 3

	var b bytes.Buffer
	w := NewEMRWriter(&b)
	for i := 0; i < numRecords; i++ {
		key := fmt.Sprintf("%d", i)
		value := fmt.Sprintf("hello %d", i)
		err := w.Emit(key, []byte(value))
		require.NoError(t, err)
	}
	w.Close()

	var i int
	r := NewEMRIterator(&b)
	for r.Next() {
		assert.Equal(t, fmt.Sprintf("%d", i), r.Key(), "key mismatch")
		assert.Equal(t, fmt.Sprintf("hello %d", i), string(r.Value()), "value mismatch")
		assert.Equal(t, 0, r.Tag())
		i++
	}
	assert.NoError(t, r.Err())
	assert.Equal(t, i, numRecords, "record count mismatch")
}

func TestEMRIteratorWithTag(t *testing.T) {
	numRecords := 3

	var b bytes.Buffer
	w := NewEMRWriter(&b)
	for i := 0; i < numRecords; i++ {
		key := fmt.Sprintf("%d", i)
		value := fmt.Sprintf("hello %d", i)
		err := w.EmitWithTag(key, i, []byte(value))
		require.NoError(t, err)
	}
	w.Close()

	var i int
	r := NewEMRIterator(&b)
	for r.Next() {
		assert.Equal(t, fmt.Sprintf("%d", i), r.Key(), "key mismatch")
		assert.Equal(t, fmt.Sprintf("hello %d", i), string(r.Value()), "value mismatch")
		assert.Equal(t, i, r.Tag())
		i++
	}
	assert.NoError(t, r.Err())
	assert.Equal(t, i, numRecords, "record count mismatch")
}
