package settings

//the compatibility layer to use new_kited's settings component in the old client

// Server returns the configured url of the backend server
func (m *Manager) Server() string {
	server, _ := m.Get(ServerKey)
	return server
}

// SetServer sets a new url for the backend server, the changes are written to disl immediately.
// Invalid URLs will be rejected with an error.
// Also, an error is returned when an error ocurred during save.
func (m *Manager) SetServer(url string) error {
	err := m.Set(ServerKey, url)
	return err
}

// stringKeyListener is an adapter to use a callback as Notifier
type stringKeyListener struct {
	watchedKey string
	callback   func(string)
}

func (s *stringKeyListener) Updated(key, value string) {
	if key == s.watchedKey {
		s.callback(value)
	}
}

func (s *stringKeyListener) Deleted(key string) {
	if s.watchedKey == key {
		s.callback("")
	}
}
