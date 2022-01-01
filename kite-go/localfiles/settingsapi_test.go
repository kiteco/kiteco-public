package localfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createFile(name string) *File {
	return &File{
		ID:      int64(1),
		Machine: "test-machine-id",
		Name:    name,
	}
}

func TestCreateFolderTree(t *testing.T) {

	// input
	input := []*File{
		createFile("/file1"),
		createFile("/folder/file2"),
	}

	// compute
	tree := CreateFolderTree(input)

	require.Equal(t, "", tree.Name)
	require.Equal(t, 1, len(tree.Folders))
	require.Equal(t, 1, len(tree.Files))
	assert.Equal(t, "file1", tree.Files[0].Name)
	require.NotNil(t, tree.Folders["folder"])
	assert.Equal(t, "folder", tree.Folders["folder"].Name)
	require.Equal(t, 1, len(tree.Folders["folder"].Files))
	assert.Equal(t, "file2", tree.Folders["folder"].Files[0].Name)
}

func TestCreateFolderTreeOldFiles(t *testing.T) {

	// input
	input := []*File{
		createFile(""),
	}

	// compute
	// just make sure nothing failed
	CreateFolderTree(input)
}

func TestWindowsNewFileSystemAPI(t *testing.T) {

	// input
	files := []*File{
		createFile("/windows/c/folder1/file1"),
		createFile("/windows/c/file2"),
		createFile("/windows/d/file3"),
		createFile("/windows/unc/domain1/folder2/file4"),
		createFile("/windows/unc/domain2/file5"),
	}

	input := CreateFolderTree(files)

	// compute
	fs := NewFileSystemAPI(input)
	assert.Equal(t, `\`, fs.Separator)

	r := fs.Root
	assert.Equal(t, "", r.Name)
	require.Equal(t, 4, len(r.Folders))
	assert.Equal(t, 0, len(r.Files))
	require.NotNil(t, r.Folders[`C:`])
	require.NotNil(t, r.Folders[`D:`])
	require.NotNil(t, r.Folders[`\\domain1`])
	require.NotNil(t, r.Folders[`\\domain2`])
	assert.Equal(t, `C:`, r.Folders[`C:`].Name)
	assert.Equal(t, `D:`, r.Folders[`D:`].Name)
	assert.Equal(t, `\\domain1`, r.Folders[`\\domain1`].Name)
	assert.Equal(t, `\\domain2`, r.Folders[`\\domain2`].Name)

	// C:
	require.Equal(t, 1, len(r.Folders[`C:`].Folders))
	require.Equal(t, 1, len(r.Folders[`C:`].Files))
	assert.Equal(t, "file2", r.Folders[`C:`].Files[0].Name)
	require.NotNil(t, r.Folders[`C:`].Folders["folder1"])
	require.Equal(t, 1, len(r.Folders[`C:`].Folders["folder1"].Files))
	assert.Equal(t, 0, len(r.Folders[`C:`].Folders["folder1"].Folders))
	assert.Equal(t, "file1", r.Folders[`C:`].Folders["folder1"].Files[0].Name)

	// D:
	require.Equal(t, 1, len(r.Folders[`D:`].Files))
	assert.Equal(t, 0, len(r.Folders[`D:`].Folders))
	assert.Equal(t, "file3", r.Folders[`D:`].Files[0].Name)

	// \\domain1
	require.Equal(t, 1, len(r.Folders[`\\domain1`].Folders))
	assert.Equal(t, 0, len(r.Folders[`\\domain1`].Files))
	require.NotNil(t, r.Folders[`\\domain1`].Folders["folder2"])
	require.Equal(t, 1, len(r.Folders[`\\domain1`].Folders["folder2"].Files))
	assert.Equal(t, 0, len(r.Folders[`\\domain1`].Folders["folder2"].Folders))
	assert.Equal(t, "file4", r.Folders[`\\domain1`].Folders["folder2"].Files[0].Name)

	// \\domain2
	require.Equal(t, 1, len(r.Folders[`\\domain2`].Files))
	assert.Equal(t, 0, len(r.Folders[`\\domain2`].Folders))
	assert.Equal(t, "file5", r.Folders[`\\domain2`].Files[0].Name)
}

func TestUnixNewFileSystemAPI(t *testing.T) {

	// input
	files := []*File{
		createFile("/folder/file1"),
		createFile("/file2"),
	}

	input := CreateFolderTree(files)

	// compute
	fs := NewFileSystemAPI(input)
	assert.Equal(t, `/`, fs.Separator)

	r := fs.Root
	assert.Equal(t, "", r.Name)
	require.Equal(t, 1, len(r.Folders))
	assert.Equal(t, 1, len(r.Files))
	require.NotNil(t, r.Folders["folder"])
	assert.Equal(t, "folder", r.Folders["folder"].Name)
	assert.Equal(t, "file2", r.Files[0].Name)
	require.Equal(t, 1, len(r.Folders["folder"].Files))
	assert.Equal(t, "file1", r.Folders["folder"].Files[0].Name)
}
