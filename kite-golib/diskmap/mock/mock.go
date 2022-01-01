package mock

import "github.com/kiteco/kiteco/kite-golib/diskmap"

type fakeMap map[string][]byte

// Empty returns an in-memory diskmap initialized with no entries.
func Empty() diskmap.Getter {
	return make(fakeMap)
}

// New returns an in-memory diskmap initialized with the provided entries.
func New(entries map[string][]byte) diskmap.Getter {
	return fakeMap(entries)
}

func (m fakeMap) Get(key string) ([]byte, error) {
	value, found := m[key]
	if found {
		return value, nil
	}
	return nil, diskmap.ErrNotFound
}

func (m fakeMap) Len() int {
	return len(m)
}
