package localfiles

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DeleteStaleContent(t *testing.T) {
	_, store, s := makeTestServer()
	defer store.FileDB.Close()

	uid := 1
	content := []byte("test data")
	hash := ComputeHash(content)
	newContent := []byte("new data")
	newHash := ComputeHash(newContent)

	// Add file to content db
	s.store.Put(int64(uid), "machine", "test.txt", content)

	// One file points to hash before delete
	f := &FileEvent{
		File: &File{
			UserID:        int64(uid),
			Machine:       "2",
			Name:          "test.txt",
			HashedContent: hash,
		},
	}
	store.Files.BatchCreateOrUpdate([]*FileEvent{f})

	// Update file contents so file will no longer point to
	// hash we want to delete
	s.store.Put(int64(uid), "machine", "test.txt", newContent)

	// ----------

	// Add file to hash db since it was deleted
	s.store.Put(int64(uid), "machine", "test.txt", content)

	// More than one file points to hash before delete
	for i := 0; i < 2; i++ {
		f = &FileEvent{
			File: &File{
				UserID:        int64(i),
				Machine:       fmt.Sprintf("%d", i+1),
				Name:          "test.txt",
				HashedContent: hash,
			},
		}
		store.Files.BatchCreateOrUpdate([]*FileEvent{f})
	}

	// Update file contents so file will no longer point to
	// hash we want to delete
	f.HashedContent = newHash
	store.Files.BatchCreateOrUpdate([]*FileEvent{f})
}

func Test_SupportedLangUploads(t *testing.T) {
	_, store, s := makeTestServer()
	defer store.FileDB.Close()

	var (
		files   FileEvents
		dbFiles []*File
		name    string
		hash    string
		err     error
		content []byte
	)

	ur := UploadRequest{
		start:   time.Now(),
		userID:  1,
		machine: "1",
	}
	contents := make(map[string]*Content)

	// Add a bunch of python files
	for i := 0; i < 10; i++ {
		name = fmt.Sprintf("test-%d.py", i)
		if i%2 == 0 {
			content = []byte("test content")
		} else {
			content = []byte(fmt.Sprintf("test content %d", i))
		}
		hash = ComputeHash(content)
		files = append(files, &FileEvent{
			File: &File{
				UserID:        ur.userID,
				Machine:       ur.machine,
				Name:          name,
				HashedContent: hash,
				CreatedAt:     time.Now(),
			},
			Content: content,
			Type:    ModifiedEvent,
		})
		contents[hash] = &Content{
			Content: content,
		}
	}

	// Make upload request
	ur.Files = files
	ur.Contents = contents
	s.uploadRequestChan <- ur

	// Wait for upload request to be processed
	ts := time.Now()
	threshold := 2 * time.Second
	for {
		dbFiles, err = store.Files.List(ur.userID, ur.machine)
		require.NoError(t, err)
		if len(dbFiles) == len(files) {
			break
		}
		if time.Since(ts) > threshold {
			t.Errorf("Expected %d files to be created, actual: %d", len(files), len(dbFiles))
			t.FailNow()
		}
	}

	// Confirm uploaded python files exist
	sort.Sort(byName(dbFiles))
	assert.Equal(t, len(dbFiles), len(files))
	for i, f := range files {
		exists, err := store.Exists(f.HashedContent)
		require.NoError(t, err)
		assert.True(t, exists, fmt.Sprintf("expected %s to exist in store", f.Name))
		assert.Equal(t, f.Name, dbFiles[i].Name, fmt.Sprintf("expected %s to exist in db", f.Name))
	}
}

func Test_UnsupportedLangUpload(t *testing.T) {
	_, store, s := makeTestServer()
	defer store.FileDB.Close()

	var (
		dbFiles []*File
		err     error
	)

	uid := int64(1)
	mid := "1"
	pyContent := []byte("py test content")
	cppContent := []byte("cpp test content")
	pyHash := ComputeHash(pyContent)
	cppHash := ComputeHash(cppContent)
	ur := UploadRequest{
		start:   time.Now(),
		userID:  uid,
		machine: mid,
		Files: FileEvents{
			&FileEvent{
				File: &File{
					UserID:        uid,
					Machine:       mid,
					Name:          "test.py",
					HashedContent: pyHash,
					CreatedAt:     time.Now(),
				},
				Content: pyContent,
				Type:    ModifiedEvent,
			},
			&FileEvent{
				File: &File{
					UserID:        uid,
					Machine:       mid,
					Name:          "test.cpp",
					HashedContent: cppHash,
					CreatedAt:     time.Now(),
				},
				Content: cppContent,
				Type:    ModifiedEvent,
			},
		},
		Contents: map[string]*Content{
			pyHash: &Content{
				Content: pyContent,
			},
			cppHash: &Content{
				Content: cppContent,
			},
		},
	}
	s.uploadRequestChan <- ur

	// Wait for upload request to be processed
	ts := time.Now()
	threshold := 2 * time.Second
	for {
		dbFiles, err = store.Files.List(ur.userID, ur.machine)
		require.NoError(t, err)
		if len(dbFiles) == 1 {
			break
		}
		if time.Since(ts) > threshold {
			t.Errorf("Expected %d files to be created, actual: %d", 1, len(dbFiles))
			t.FailNow()
		}
	}

	// Confirm uploaded python files exist and non-python files do not
	assert.Equal(t, len(dbFiles), 1)
	f := dbFiles[0]
	assert.Equal(t, "test.py", f.Name)
	exists, err := store.Exists(pyHash)
	require.NoError(t, err)
	assert.True(t, exists, fmt.Sprintf("expected %s to exist in store", pyHash))
	exists, err = store.Exists(cppHash)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "file not found"))
}

func Test_MatchingContent(t *testing.T) {
	_, store, s := makeTestServer()
	defer store.FileDB.Close()

	var (
		dbFiles []*File
		err     error
	)

	uid := int64(1)
	mid := "1"
	content := []byte("test content")
	hash := ComputeHash(content)
	ur := UploadRequest{
		start:   time.Now(),
		userID:  uid,
		machine: mid,
		Files: FileEvents{
			&FileEvent{
				File: &File{
					UserID:        uid,
					Machine:       mid,
					Name:          "test.py",
					HashedContent: hash,
					CreatedAt:     time.Now(),
				},
				Content: content,
				Type:    ModifiedEvent,
			},
			&FileEvent{
				File: &File{
					UserID:        uid,
					Machine:       mid,
					Name:          "test.cpp",
					HashedContent: hash,
					CreatedAt:     time.Now(),
				},
				Content: content,
				Type:    ModifiedEvent,
			},
		},
		Contents: map[string]*Content{
			hash: &Content{
				Content: content,
			},
		},
	}
	s.uploadRequestChan <- ur

	// Wait for upload request to be processed
	ts := time.Now()
	threshold := 2 * time.Second
	for {
		dbFiles, err = store.Files.List(ur.userID, ur.machine)
		require.NoError(t, err)
		if len(dbFiles) == 1 {
			break
		}
		if time.Since(ts) > threshold {
			t.Errorf("Expected %d files to be created, actual: %d", 1, len(dbFiles))
			t.FailNow()
		}
	}

	// Confirm uploaded python files exist and non-python files do not
	assert.Equal(t, len(dbFiles), 1)
	f := dbFiles[0]
	assert.Equal(t, "test.py", f.Name)
	exists, err := store.Exists(hash)
	require.NoError(t, err)
	assert.True(t, exists, fmt.Sprintf("expected %s to exist in store", hash))
}

func Test_Remove(t *testing.T) {
	_, store, s := makeTestServer()
	defer store.FileDB.Close()

	var (
		files   FileEvents
		dbFiles []*File
		name    string
		hash    string
		err     error
		content []byte
	)

	numFiles := 10
	ur := UploadRequest{
		start:   time.Now(),
		userID:  1,
		machine: "1",
	}
	contents := make(map[string]*Content)

	// Add a bunch of python files
	for i := 0; i < numFiles; i++ {
		name = fmt.Sprintf("test-%d.py", i)
		if i%2 == 0 {
			content = []byte("test content")
		} else {
			content = []byte(fmt.Sprintf("test content %d", i))
		}
		hash = ComputeHash(content)
		files = append(files, &FileEvent{
			File: &File{
				UserID:        ur.userID,
				Machine:       ur.machine,
				Name:          name,
				HashedContent: hash,
				CreatedAt:     time.Now(),
			},
			Content: content,
			Type:    ModifiedEvent,
		})
		contents[hash] = &Content{
			Content: content,
		}
	}

	// Make upload request
	ur.Files = files
	ur.Contents = contents
	s.uploadRequestChan <- ur

	// Wait for upload request to be processed
	ts := time.Now()
	threshold := 2 * time.Second
	for {
		dbFiles, err = store.Files.List(ur.userID, ur.machine)
		require.NoError(t, err)
		if len(dbFiles) == len(files) {
			break
		}
		if time.Since(ts) > threshold {
			t.Errorf("Expected %d files to be created, actual: %d", len(files), len(dbFiles))
			t.FailNow()
		}
	}

	// Confirm uploaded python files exist and non-python files do not
	sort.Sort(byName(dbFiles))
	assert.Equal(t, len(dbFiles), len(files))
	for i, f := range files {
		exists, err := store.Exists(f.HashedContent)
		require.NoError(t, err)
		assert.True(t, exists, fmt.Sprintf("expected %s to exist in store", f.Name))
		assert.Equal(t, f.Name, dbFiles[i].Name, fmt.Sprintf("expected %s to exist in db", f.Name))
	}

	ur = UploadRequest{
		start:    time.Now(),
		userID:   1,
		machine:  "1",
		Contents: make(map[string]*Content),
	}
	removedFiles := []*FileEvent{}
	contents = make(map[string]*Content)
	for i := range files {
		if i%3 != 0 {
			continue
		}
		f := files[i]
		f.Type = RemovedEvent
		removedFiles = append(removedFiles, f)
	}
	ur.Files = removedFiles
	ur.Contents = contents
	s.uploadRequestChan <- ur

	// Wait for upload request to be processed
	ts = time.Now()
	for {
		dbFiles, err = store.Files.List(ur.userID, ur.machine)
		require.NoError(t, err)
		if len(dbFiles) == numFiles-len(removedFiles) {
			break
		}
		if time.Since(ts) > threshold {
			t.Errorf("Expected %d files to exist after deletions, actual: %d", numFiles-len(removedFiles), len(dbFiles))
			t.FailNow()
		}
	}

	// Confirm deleted files do not exist in file db
	assert.Equal(t, len(dbFiles), numFiles-len(removedFiles))
	for _, f := range removedFiles {
		exists, err := store.Exists(f.HashedContent)
		require.NoError(t, err)
		// Hashes are not deleted from the file store, only from the file db
		assert.True(t, exists, fmt.Sprintf("expected %s to exist in store", f.Name))
		exists = false
		for _, f2 := range dbFiles {
			if f.Name == f2.Name {
				exists = true
				break
			}
		}
		assert.False(t, exists, fmt.Sprintf("expected %s to not exist in db", f.Name))
	}
}

func Test_MissingContent(t *testing.T) {
	_, store, s := makeTestServer()
	defer store.FileDB.Close()

	var (
		dbFiles []*File
		err     error
	)

	uid := 1
	mid := "1"
	content1 := []byte("test content")
	hash1 := ComputeHash(content1)
	content2 := []byte("missing content")
	hash2 := ComputeHash(content2)
	ur := UploadRequest{
		start:   time.Now(),
		userID:  int64(uid),
		machine: mid,
		Files: FileEvents{
			&FileEvent{
				File: &File{
					UserID:        int64(uid),
					Machine:       mid,
					Name:          "test1.py",
					HashedContent: hash1,
					CreatedAt:     time.Now(),
				},
				Content: content1,
				Type:    ModifiedEvent,
			},
			&FileEvent{
				File: &File{
					UserID:        int64(uid),
					Machine:       mid,
					Name:          "test2.py",
					HashedContent: hash2,
					CreatedAt:     time.Now(),
				},
				Content: content2,
				Type:    ModifiedEvent,
			},
		},
		Contents: map[string]*Content{
			hash1: &Content{
				Content: content1,
			},
		},
	}
	s.uploadRequestChan <- ur

	// Wait for upload request to be processed
	ts := time.Now()
	threshold := 2 * time.Second
	for {
		dbFiles, err = store.Files.List(ur.userID, ur.machine)
		require.NoError(t, err)
		if len(dbFiles) == 1 {
			break
		}
		if time.Since(ts) > threshold {
			t.Errorf("Expected %d files to be created, actual: %d", 1, len(dbFiles))
			t.FailNow()
		}
	}

	// Confirm file with content in request exists and file missing content does not
	assert.Equal(t, len(dbFiles), 1)
	f := dbFiles[0]
	assert.Equal(t, "test1.py", f.Name)
	exists, err := store.Exists(hash1)
	require.NoError(t, err)
	assert.True(t, exists, fmt.Sprintf("expected %s to exist in store", hash1))
	_, err = store.Exists(hash2)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not found"), fmt.Sprintf("expected %s not to exist in store", hash2))
}

// --

type byName []*File

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return b[i].Name < b[j].Name }
