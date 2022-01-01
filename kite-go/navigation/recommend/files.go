package recommend

import (
	"errors"
	"sync"

	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
)

type fileID int

type fileIndex struct {
	root  localpath.Absolute
	index map[localpath.Relative]fileID
	files []localpath.Relative
	m     *sync.Mutex
}

func (r recommender) newFileIndex() *fileIndex {
	return &fileIndex{
		root:  r.opts.Root,
		index: make(map[localpath.Relative]fileID),
		m:     new(sync.Mutex),
	}
}

func (f *fileIndex) toID(path localpath.Absolute) (fileID, error) {
	f.m.Lock()
	defer f.m.Unlock()

	rel, err := path.RelativeTo(f.root)
	if err != nil {
		return 0, err
	}
	if id, ok := f.index[rel]; ok {
		return id, nil
	}
	id := fileID(len(f.files))
	f.files = append(f.files, rel)
	f.index[rel] = id
	return id, nil
}

func (f fileIndex) fromID(id fileID) (localpath.Absolute, error) {
	f.m.Lock()
	defer f.m.Unlock()

	if int(id) > len(f.files) {
		return "", errors.New("invalid fileID")
	}
	return f.root.Join(f.files[id]), nil
}
