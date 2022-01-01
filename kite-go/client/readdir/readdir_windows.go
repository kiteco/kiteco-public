// +build windows

package readdir

// List returns the entries of the given directory
// TODO(tarak): optimize this
func List(path string) []Dirent {
	names, err := NamesUnsorted(path)
	if err != nil {
		return nil
	}

	var ret []Dirent
	for _, s := range names {
		ret = append(ret, Dirent{
			Path:         s,
			IsDir:        false,
			DTypeEnabled: false,
		})
	}

	return ret
}
