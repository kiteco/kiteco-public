package readdir

import "os"

// Dirent defines properties of a directory entry
type Dirent struct {
	Path         string
	IsDir        bool
	DTypeEnabled bool
	Info         os.FileInfo
}

// NamesUnsorted is a copy of filepath.readDirNames that does not sort the returned slice
func NamesUnsorted(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	return names, nil
}
