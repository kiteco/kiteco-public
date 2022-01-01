package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type sessionID int

const (
	maxSessionCount = 10
	// how long to wait after receiving a request before timing out a session
	sessionTimeout = 5 * time.Minute
)

type sessionMgr struct {
	sessions map[sessionID]*session
	res      *resources
	m        sync.Mutex
}

func newSessionMgr(res *resources) *sessionMgr {
	s := &sessionMgr{
		sessions: make(map[sessionID]*session),
		res:      res,
	}

	ticker := time.NewTicker(1 * time.Second)

	go func() {
		for range ticker.C {
			s.checkExpiredSessions()
		}
	}()

	return s
}

// CreateSession for the specified request, we hand in the initial request for debugging purporses
func (s *sessionMgr) CreateSession(builder *builder) (*session, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if len(s.sessions) >= maxSessionCount {
		// evict the oldest session
		var oldest *session
		for _, session := range s.sessions {
			if session != nil && (oldest == nil || session.lastRequest.Before(oldest.lastRequest)) {
				oldest = session
			}
		}

		if oldest == nil {
			return nil, fmt.Errorf("max session count exceeded, no sessions to evict")
		}

		log.Printf("will evict oldest session ID %d", oldest.id)
		delete(s.sessions, oldest.id)
		oldest.Cleanup()
	}

	id := s.newSessionID()
	session := newSession(id, builder)
	s.sessions[id] = session

	return session, nil
}

func (s *sessionMgr) GetSession(id sessionID) *session {
	s.m.Lock()
	defer s.m.Unlock()

	return s.sessions[id]
}

func (s *sessionMgr) Stats() map[sessionID]sessionStats {
	s.m.Lock()
	defer s.m.Unlock()

	stats := make(map[sessionID]sessionStats)

	for id, session := range s.sessions {
		if session == nil {
			continue
		}
		stats[id] = session.Stats()
	}

	return stats
}

func (s *sessionMgr) Sessions() map[sessionID]*session {
	s.m.Lock()
	defer s.m.Unlock()

	m := make(map[sessionID]*session, len(s.sessions))

	for id, s := range s.sessions {
		m[id] = s
	}

	return m
}

// newSessionID returns the session ID as the UNIX millisecond timestamp
func (s *sessionMgr) newSessionID() sessionID {
	var id sessionID

	id = sessionID(time.Now().UnixNano() / 1000000)

	// avoid collisions
	for {
		_, found := s.sessions[id]
		if !found {
			break
		}
		id++
	}

	return id
}

func (s *sessionMgr) checkExpiredSessions() {
	s.m.Lock()
	defer s.m.Unlock()

	for _, session := range s.sessions {
		if session == nil {
			continue
		}

		if session.Working() {
			continue
		}

		if time.Since(session.LastRequest()) > sessionTimeout {
			log.Printf("session %d expired", session.id)
			delete(s.sessions, session.id)
			session.Cleanup()
		}
	}
}

func (s *sessionMgr) Kill(id sessionID) error {
	s.m.Lock()
	defer s.m.Unlock()

	session := s.sessions[id]
	if session == nil {
		return fmt.Errorf("session %d not found", id)
	}

	delete(s.sessions, id)

	session.Cleanup()

	return nil
}

type session struct {
	id      sessionID
	builder *builder

	// keep track of the last request so we know when to time out the session
	lastRequest time.Time

	lastReqLock sync.Mutex
	// Only one thread should be able to get a batch at a time for this session.
	getBatchLock sync.Mutex

	working bool
}

func newSession(id sessionID, builder *builder) *session {
	return &session{
		id:          id,
		builder:     builder,
		lastRequest: time.Now(),
	}
}

func (s *session) GetBatch() ([]*sample, error) {
	s.lastReqLock.Lock()
	s.lastRequest = time.Now()
	s.working = true
	s.lastReqLock.Unlock()

	s.getBatchLock.Lock()
	defer s.getBatchLock.Unlock()

	samples, err := s.builder.GetBatch()

	s.lastReqLock.Lock()
	s.lastRequest = time.Now()
	s.working = false
	s.lastReqLock.Unlock()

	return samples, err
}

func (s *session) Ping() {
	s.lastReqLock.Lock()
	defer s.lastReqLock.Unlock()
	s.lastRequest = time.Now()
}

func (s *session) Working() bool {
	s.lastReqLock.Lock()
	defer s.lastReqLock.Unlock()
	return s.working
}

func (s *session) LastRequest() time.Time {
	s.lastReqLock.Lock()
	defer s.lastReqLock.Unlock()
	return s.lastRequest
}

func (s *session) Cleanup() {
	s.builder.Cleanup()
}

type sessionStats struct {
	BuildCount    int64
	BuildErrCount int64
	BuildOKCount  int64

	LastRequest time.Time
}

func (s *session) Stats() sessionStats {
	return sessionStats{
		BuildCount:    atomic.LoadInt64(&s.builder.buildCount),
		BuildErrCount: atomic.LoadInt64(&s.builder.buildErrCount),
		BuildOKCount:  atomic.LoadInt64(&s.builder.BuildOKCount),
		LastRequest:   s.LastRequest(),
	}
}
