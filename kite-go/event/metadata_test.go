package event

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Add(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	start := time.Now().UnixNano()
	m := &Metadata{
		Stream:   "test",
		UserID:   1,
		Start:    start,
		End:      start + 10,
		Count:    10,
		Size:     100,
		Filename: "foo/bar",
	}

	if err := mm.Add(m); err != nil {
		t.Error("expected metadata to be added, got:", err)
	}
}

func Test_Get(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	initTime := time.Now().UnixNano()
	// Since blocks are written in reverse time order,
	// end starts at the oldest time (initTime) and increases.
	end := initTime
	start := end + 10

	for i := 0; i < 10; i++ {
		m := &Metadata{
			Stream:   "test",
			UserID:   1,
			Start:    start,
			End:      end,
			Count:    10,
			Size:     100,
			Filename: fmt.Sprintf("block_%d", i),
		}
		if err := mm.Add(m); err != nil {
			t.Error("expected metadata to be added, got:", err)
		}
		end = start + 1
		start = end + 10
	}

	// Latest should return all metadata
	blocks, err := mm.Latest(10, 1, "test")
	if err != nil {
		t.Error("expected get to succeed, got:", err)
	}
	assert.Len(t, blocks, 10, "The length of blocks is not 10")

	// ts after end should return all the metadata
	blocks, err = mm.Get(10, 1, "test", end+1)
	if err != nil {
		t.Error("expected get to succeed, got:", err)
	}
	assert.Len(t, blocks, 10, "The length of blocks is not 10")

	// ts after initial time + 20 should return the first 2 metadatas
	blocks, err = mm.Get(10, 1, "test", initTime+20)
	if err != nil {
		t.Error("expected get to succeed, got:", err)
	}
	assert.Len(t, blocks, 2, "The length of blocks is not 2")

	// ts after initial time + 32 should return the first 3 metadatas
	blocks, err = mm.Get(10, 1, "test", initTime+32)
	if err != nil {
		t.Error("expected get to succeed, got:", err)
	}
	assert.Len(t, blocks, 3, "The length of blocks is not 3")

	// Latest should return one batch of metadatas
	blocks, err = mm.Latest(2, 1, "test")
	if err != nil {
		t.Error("expected get to succeed, got:", err)
	}
	assert.Len(t, blocks, 2, "The length of blocks is not 2")
}

func Test_AddDuplicates(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	start := time.Now().UnixNano()
	m := &Metadata{
		Stream:   "test",
		UserID:   1,
		Start:    start,
		End:      start + 10,
		Count:    10,
		Size:     100,
		Filename: "foo/bar",
	}

	for i := 0; i < 5; i++ {
		if err := mm.Add(m); err != nil {
			t.Error("expected metadata to be added, got:", err)
		}
	}

	// Latest should return one block since the rest are duplicates.
	blocks, err := mm.Latest(10, 1, "test")
	if err != nil {
		t.Error("expected get to succeed, got:", err)
	}
	assert.Len(t, blocks, 1, "The length of blocks is not 1")
}

func Test_DeleteNotExists(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	err := mm.Delete(1)
	assert.Nil(t, err, "expected delete for metadata that does not exist to succeed")
}

func Test_DeleteExists(t *testing.T) {
	mm := createTestManager()
	defer mm.db.Close()

	initTime := time.Now().UnixNano()
	// Since blocks are written in reverse time order,
	// end starts at the oldest time (initTime) and increases.
	end := initTime
	start := end + 10

	var metadata []*Metadata
	for i := 0; i < 10; i++ {
		m := &Metadata{
			Stream:   "test",
			UserID:   1,
			Start:    start,
			End:      end,
			Count:    10,
			Size:     100,
			Filename: fmt.Sprintf("block_%d", i),
		}
		if err := mm.Add(m); err != nil {
			t.Error("expected metadata to be added, got:", err)
		}
		metadata = append(metadata, m)
		end = start + 1
		start = end + 10
	}

	err := mm.Delete(metadata[5].ID)
	assert.Nil(t, err, "expected delete for metadata that exists to succeed")

	// Latest should return one block since the rest are duplicates.
	blocks, err := mm.Latest(10, 1, "test")
	if err != nil {
		t.Error("expected get to succeed, got:", err)
	}
	assert.Len(t, blocks, 9, "The length of blocks is not 9")
}
