package localfiles

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BatchCreateOrUpdate(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	var files []*FileEvent

	// Should not return an error when files is empty.
	err := store.Files.BatchCreateOrUpdate(files)
	assert.Nil(t, err, "expected batch create or update to succeed")

	// Create ten users, each with two machines and two files per machine.
	s := "test data"
	for i := 1; i < 11; i++ {
		for j := 1; j < 3; j++ {
			for k := 1; k < 3; k++ {
				uid := int64(i)
				mid := fmt.Sprintf("%d", j)
				content := []byte(fmt.Sprintf("%s_%d", s, k))
				filename := fmt.Sprintf("%d_%d_%d.txt", i, j, k)

				f := &FileEvent{
					File: &File{
						UserID:        uid,
						Machine:       mid,
						Name:          filename,
						HashedContent: ComputeHash(content),
					},
				}
				files = append(files, f)
			}
		}
	}

	// Add the new files.
	err = store.Files.BatchCreateOrUpdate(files)
	assert.Nil(t, err, "expected batch create or update to succeed")

	// Confirm all the new files were created.
	for i, f1 := range files {
		id := int64(i + 1)
		f2, err := store.Files.Get(id)
		assert.Nil(t, err, "expected file get to succeed")
		assert.Equal(t, f1.UserID, f2.UserID)
		assert.Equal(t, f1.Machine, f2.Machine)
		assert.Equal(t, f1.Name, f2.Name)
		assert.Equal(t, f1.HashedContent, f2.HashedContent)

		// Update half of the users' existing files
		if i%2 == 0 {
			content := []byte(fmt.Sprintf("%s_%d_%s", s, i, s))
			hash := ComputeHash(content)
			assert.NotEqual(t, f1.HashedContent, hash)
			f1.HashedContent = hash
		}
	}

	// Add the files, half of which have been updated, the rest remain unchanged.
	err = store.Files.BatchCreateOrUpdate(files)
	assert.Nil(t, err, "expected batch create or update to succeed")

	// The stored files should match the updated and unchanged files.
	for i, f1 := range files {
		id := int64(i + 1)
		f2, err := store.Files.Get(id)
		assert.Nil(t, err, "expected file get to succeed")
		assert.Equal(t, f1.UserID, f2.UserID)
		assert.Equal(t, f1.Machine, f2.Machine)
		assert.Equal(t, f1.Name, f2.Name)
		assert.Equal(t, f1.HashedContent, f2.HashedContent)
	}
}

func Test_Get(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	content := []byte("test data")

	// File does not exist
	_, err := store.Files.Get(1)
	assert.Equal(t, sql.ErrNoRows, err)

	// File exists
	err = store.Put(1, "machine", "test.txt", content)
	assert.Nil(t, err, "expected put to succeed")

	file, err := store.Files.Get(1)
	assert.Nil(t, err, "expected file get to succeed")
	assert.EqualValues(t, 1, file.UserID)
	assert.Equal(t, "machine", file.Machine)
	assert.Equal(t, "test.txt", file.Name)
	assert.Equal(t, ComputeHash(content), file.HashedContent)
}

func Test_Delete(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	content := []byte("test data")
	f := &FileEvent{
		File: &File{
			UserID:        1,
			Machine:       "2",
			Name:          "test.txt",
			HashedContent: ComputeHash(content),
		},
	}

	// Deleting a file that does not exist should not return an error.
	err := store.Files.Delete(f)
	assert.Nil(t, err, "expected file delete to succeed")

	// Delete a file that does exist.
	err = store.Put(f.UserID, f.Machine, f.Name, content)
	assert.Nil(t, err, "expected put to succeed")
	err = store.Files.Delete(f)
	assert.Nil(t, err, "expected file delete to succeed")
	// Getting deleted file should return an error.
	_, err = store.Files.Get(1)
	assert.Equal(t, sql.ErrNoRows, err)
}

func Test_DeleteUser(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	// Create two users, each with two machines and two files per machine.
	numFiles := 2
	numMachines := 2
	s := "test data"
	for i := 1; i < 3; i++ {
		uid := int64(i)
		for j := 1; j <= numMachines; j++ {
			mid := fmt.Sprintf("%d", j)
			for k := 1; k <= numFiles; k++ {
				content := []byte(fmt.Sprintf("%s_%d", s, k))
				filename := fmt.Sprintf("%d_%d_%d.txt", i, j, k)

				err := store.Put(uid, mid, filename, content)
				assert.Nil(t, err, "expected file create to succeed")
			}

			// Confirm all files exist for user
			files, err := store.Files.List(int64(uid), mid)
			assert.Nil(t, err, "expected list to succeed")
			assert.Len(t, files, numFiles)
		}
	}

	// DeleteUser should result in 0 files existing for user 1
	err := store.Files.DeleteUser(1)
	assert.Nil(t, err, "expected delete all to succeed")
	for j := 1; j <= numMachines; j++ {
		mid := fmt.Sprintf("%d", j)
		files, err := store.Files.List(1, mid)
		assert.Nil(t, err, "expected list to succeed")
		assert.Len(t, files, 0)
	}

	// User 2 should still have all 4 files.
	for j := 1; j <= numMachines; j++ {
		mid := fmt.Sprintf("%d", j)
		files, err := store.Files.List(2, mid)
		assert.Nil(t, err, "expected list to succeed")
		assert.Len(t, files, numFiles)
	}
}

func Test_DeleteUserMachine(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	// Create two users, each with two machines and two files per machine.
	numFiles := 2
	s := "test data"
	for i := 1; i < 3; i++ {
		uid := int64(i)
		for j := 1; j < 3; j++ {
			mid := fmt.Sprintf("%d", j)
			for k := 1; k <= numFiles; k++ {
				content := []byte(fmt.Sprintf("%s_%d", s, k))
				filename := fmt.Sprintf("%d_%d_%d.txt", i, j, k)

				err := store.Put(uid, mid, filename, content)
				assert.Nil(t, err, "expected file create to succeed")
			}

			// Confirm all files exist for user
			files, err := store.Files.List(int64(uid), mid)
			assert.Nil(t, err, "expected list to succeed")
			assert.Len(t, files, numFiles)
		}
	}

	// DeleteUserMachine should result in 0 files for user 2, machine 1
	err := store.Files.DeleteUserMachine(2, "1")
	assert.Nil(t, err, "expected delete all to succeed")
	files, err := store.Files.List(2, "1")
	assert.Nil(t, err, "expected list to succeed")
	assert.Len(t, files, 0)

	// User 2's files on other machines should still exist.
	files, err = store.Files.List(2, "2")
	assert.Nil(t, err, "expected list to succeed")
	assert.Len(t, files, numFiles)
}

func Test_List(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	type expectedFile struct {
		UserID        int64
		Machine       string
		Name          string
		HashedContent string
	}

	// List should return 0 files when empty.
	files, err := store.Files.List(1, "1")
	assert.Nil(t, err, "expected list to succeed")
	assert.Len(t, files, 0)

	testFiles := make(map[string]expectedFile)

	// Create two users, each with two machines and two files per machine.
	numFiles := 2
	s := "test data"
	for i := 1; i < 3; i++ {
		uid := int64(i)
		for j := 1; j < 3; j++ {
			mid := fmt.Sprintf("%d", j)
			for k := 1; k <= numFiles; k++ {
				content := []byte(fmt.Sprintf("%s_%d", s, k))
				filename := fmt.Sprintf("%d_%d_%d.txt", i, j, k)

				err := store.Put(uid, mid, filename, content)
				assert.Nil(t, err, "expected file create to succeed")

				testFiles[filename] = expectedFile{
					UserID:        int64(i),
					Machine:       mid,
					Name:          filename,
					HashedContent: ComputeHash(content),
				}

				// List should return k files for user
				files, err := store.Files.List(uid, mid)
				assert.Nil(t, err, "expected list to succeed")
				assert.Len(t, files, k)

				// Contents should match
				for _, f := range files {
					tf := testFiles[f.Name]
					assert.Equal(t, f.UserID, tf.UserID)
					assert.Equal(t, f.Machine, tf.Machine)
					assert.Equal(t, f.Name, tf.Name)
					assert.Equal(t, f.HashedContent, tf.HashedContent)
				}
			}
		}
	}
}

func Test_ListChan(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	type expectedFile struct {
		UserID        int64
		Machine       string
		Name          string
		HashedContent string
	}

	// List should return 0 files when empty.
	fileChan := store.Files.ListChan(context.Background(), -1, "")
	assert.Len(t, fileChan, 0)
	var files []*File
	for f := range fileChan {
		files = append(files, f)
	}
	assert.Len(t, files, 0)

	testFiles := make(map[string]expectedFile)

	// Create two users, each with two machines and two files per machine.
	numFiles := 2
	s := "test data"
	for i := 1; i < 3; i++ {
		uid := int64(i)
		for j := 1; j < 3; j++ {
			mid := fmt.Sprintf("%d", j)
			for k := 1; k <= numFiles; k++ {
				content := []byte(fmt.Sprintf("%s_%d", s, k))
				filename := fmt.Sprintf("%d_%d_%d.txt", i, j, k)

				err := store.Put(uid, mid, filename, content)
				assert.Nil(t, err, "expected file create to succeed")

				testFiles[filename] = expectedFile{
					UserID:        int64(i),
					Machine:       mid,
					Name:          filename,
					HashedContent: ComputeHash(content),
				}

				// ListChan should return k files for user
				fileChan := store.Files.ListChan(context.Background(), uid, mid)
				var files []*File
				for f := range fileChan {
					files = append(files, f)
				}
				assert.Len(t, files, k)
			}

			// User should have two files per machine
			fileChan := store.Files.ListChan(context.Background(), uid, mid)
			var files []*File
			for f := range fileChan {
				files = append(files, f)
				tf := testFiles[f.Name]
				assert.Equal(t, f.UserID, tf.UserID)
				assert.Equal(t, f.Machine, tf.Machine)
				assert.Equal(t, f.Name, tf.Name)
				assert.Equal(t, f.HashedContent, tf.HashedContent)
			}
			assert.Len(t, files, numFiles)
		}
	}
}

func Test_Machines(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	// Create two users, each with two machines and two files per machine.
	s := "test data"
	for i := 1; i < 3; i++ {
		uid := int64(i)

		// User has no machines
		machines, err := store.Files.Machines(uid)
		assert.Nil(t, err, "expected machines to succeed")
		assert.Len(t, machines, 0)

		for j := 1; j < 3; j++ {
			mid := fmt.Sprintf("%d", j)
			for k := 1; k < 3; k++ {
				content := []byte(fmt.Sprintf("%s_%d", s, k))
				filename := fmt.Sprintf("%d_%d_%d.txt", i, j, k)
				err = store.Put(uid, mid, filename, content)
			}
			// User has j machines.
			machines, err = store.Files.Machines(uid)
			assert.Nil(t, err, "expected machines to succeed")
			assert.Len(t, machines, j)
		}
	}
}

func Test_EscapeName(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	f1 := &FileEvent{
		File: &File{
			UserID:        1,
			Machine:       "1",
			Name:          "user-`test's.py",
			HashedContent: ComputeHash([]byte("test")),
		},
	}

	// Add the new files.
	err := store.Files.BatchCreateOrUpdate([]*FileEvent{f1})
	assert.Nil(t, err, "expected batch create or update to succeed")

	f2, err := store.Files.Get(1)
	assert.Nil(t, err, "expected file get to succeed")
	assert.Equal(t, f1.UserID, f2.UserID)
	assert.Equal(t, f1.Machine, f2.Machine)
	assert.Equal(t, f1.Name, f2.Name)
	assert.Equal(t, f1.HashedContent, f2.HashedContent)
}

func Test_LongFilename(t *testing.T) {
	store := makeInMemoryTestStore()
	defer store.FileDB.Close()

	name := make([]rune, 256)
	for i := range name {
		name[i] = 'a'
	}

	// Name 1 char over limit should be filtered
	f := &FileEvent{
		File: &File{
			UserID:        1,
			Machine:       "1",
			Name:          string(name),
			HashedContent: ComputeHash([]byte("test")),
		},
	}
	err := store.Files.BatchCreateOrUpdate([]*FileEvent{f})
	assert.Nil(t, err, "expected batch create or update to succeed")
	_, err = store.Files.Get(1)
	assert.NotNil(t, err, "expected file get to fail")

	// Name with max chars should succeed
	f = &FileEvent{
		File: &File{
			UserID:        1,
			Machine:       "1",
			Name:          string(name[1:]),
			HashedContent: ComputeHash([]byte("test")),
		},
	}
	err = store.Files.BatchCreateOrUpdate([]*FileEvent{f})
	assert.Nil(t, err, "expected batch create or update to succeed")
	_, err = store.Files.Get(1)
	assert.Nil(t, err, "expected file get to succeed")
}
