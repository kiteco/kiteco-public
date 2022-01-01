package jsonutil

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Number struct {
	X int
}

func decodeAll(t *testing.T, s string, filename string, handler interface{}) error {
	tempdir, _ := ioutil.TempDir("", "")
	defer os.RemoveAll(tempdir)
	temppath := filepath.Join(tempdir, filename)
	err := ioutil.WriteFile(temppath, []byte(s), 0777)
	require.NoError(t, err)
	return DecodeAllFrom(temppath, handler)
}

func TestDecodeAll_NoReturn(t *testing.T) {
	s := `{"X": 1}{"X": 2}{"x": 3}`
	var numbers []int
	err := decodeAll(t, s, "xyz.json", func(num *Number) {
		numbers = append(numbers, num.X)
	})

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, numbers)
}

func TestDecodeAll_WithReturn(t *testing.T) {
	s := `{"X": 1}{"X": 2}{"x": 3}`
	var numbers []int
	err := decodeAll(t, s, "xyz.json", func(num *Number) error {
		numbers = append(numbers, num.X)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, numbers)
}

func TestDecodeAll_Stop(t *testing.T) {
	s := `{"X": 1}{"X": 2}{"x": 3}`
	var n int
	err := decodeAll(t, s, "xyz.json", func(num *Number) error {
		n++
		return ErrStop
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestDecodeAll_Abort(t *testing.T) {
	s := `{"X": 1}{"X": 2}{"x": 3}`
	var n int
	err := decodeAll(t, s, "xyz.json", func(num *Number) error {
		n++
		return errors.New("abort")
	})

	assert.Error(t, err)
	assert.Equal(t, 1, n)
}

func TestDecodeAll_Malformed(t *testing.T) {
	s := `{"X": 1}{foo`
	var numbers []int
	err := decodeAll(t, s, "xyz.json", func(num *Number) error {
		numbers = append(numbers, num.X)
		return nil
	})

	assert.Error(t, err)
	assert.Len(t, numbers, 1)
}

func TestDecodeAllFrom(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "numbers.json.gz")
	f, err := os.Create(path)
	require.NoError(t, err)

	w := gzip.NewWriter(f)
	e := json.NewEncoder(w)
	require.NoError(t, e.Encode(&Number{1}))
	require.NoError(t, e.Encode(&Number{2}))
	require.NoError(t, e.Encode(&Number{3}))

	w.Close()
	f.Close()

	var numbers []int
	err = DecodeAllFrom(path, func(num *Number) {
		numbers = append(numbers, num.X)
	})

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, numbers)
}
