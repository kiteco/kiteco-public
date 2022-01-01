package localfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ContentPut(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	uid := 1
	content := []byte("test data")
	err := store.Put(int64(uid), "machine", "/some/path", content)
	assert.Nil(t, err, "expected create to succeed")
}

func Test_ContentGet(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	uid := 1
	content := []byte("test data")
	hash := ComputeHash(content)

	// Content does not exist
	value, err := store.Get(hash)
	assert.NotNil(t, err, "expected get to fail")

	// Content exists
	store.Put(int64(uid), "machine", "/some/path", content)
	value, err = store.Get(hash)
	assert.Nil(t, err, "expected get to succeed")
	assert.Equal(t, string(content), string(value))
}

func Test_Migrate(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	err := store.Migrate()
	assert.Nil(t, err, "expected migrate to succeed")
	// Multiple calls to migrate should succeed
	err = store.Migrate()
	assert.Nil(t, err, "expected migrate to succeed")
}
