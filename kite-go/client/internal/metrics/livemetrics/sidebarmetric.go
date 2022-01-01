package livemetrics

import (
	"sync"
)

// sidebarSumMetrics gathers up data and sums it over the interval between kite_status events
type sidebarSumMetrics struct {
	data map[string]int
	mu   sync.Mutex
}

func newSidebarSumMetrics() *sidebarSumMetrics {
	s := &sidebarSumMetrics{}
	s.data = make(map[string]int)
	return s
}

func (s *sidebarSumMetrics) zero() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]int)
}

func (s *sidebarSumMetrics) dump() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.data
	s.data = make(map[string]int)
	return n
}

func (s *sidebarSumMetrics) update(operand map[string]int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range operand {
		s.data[k] += v
	}
}

// sidebarMostRecentMetrics gathers up data sent from the sidebar and only uses the most
// recent event sent
type sidebarMostRecentMetrics struct {
	data map[string]int
	mu   sync.Mutex
}

func newSidebarMostRecentMetrics() *sidebarMostRecentMetrics {
	s := &sidebarMostRecentMetrics{}
	s.data = make(map[string]int)
	return s
}

func (s *sidebarMostRecentMetrics) zero() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]int)
}

func (s *sidebarMostRecentMetrics) dump() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.data
	s.data = make(map[string]int)
	return n
}

func (s *sidebarMostRecentMetrics) update(update map[string]int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range update {
		s.data[k] = v
	}
}
