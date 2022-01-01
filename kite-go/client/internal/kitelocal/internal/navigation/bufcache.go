package navigation

import "sync"

type bufferCache struct {
	m        sync.Mutex
	path     string
	contents string
}

func (a *bufferCache) update(path, contents string) {
	a.m.Lock()
	defer a.m.Unlock()

	a.path = path
	a.contents = contents
}

func (a *bufferCache) bytes(path string) []byte {
	a.m.Lock()
	defer a.m.Unlock()

	if path != a.path {
		return nil
	}
	return []byte(a.contents)
}
