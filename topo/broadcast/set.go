package broadcast

import (
	"sync"
	"time"
)

type ExpiringSet struct {
	slice []element
	set   map[string]bool
	ttl   time.Duration
	mutex sync.RWMutex
}

type element struct {
	t time.Time
	v string
}

func NewExpiringSet(ttl time.Duration) *ExpiringSet {
	return &ExpiringSet{
		slice: make([]element, 0),
		set:   make(map[string]bool),
		ttl:   ttl,
	}
}

func (s *ExpiringSet) Add(v string) {
	s.mutex.Lock()
	s.slice = append(s.slice, element{time.Now(), v})
	s.set[v] = true
	if len(s.slice) == 1 {
		go s.remover()
	}
	s.mutex.Unlock()
}

func (s *ExpiringSet) Has(v string) bool {
	s.mutex.RLock()
	_, has := s.set[v]
	s.mutex.RUnlock()
	return has
}

func (s *ExpiringSet) remover() {
	for {
		s.mutex.RLock()
		timer := time.After(s.ttl - time.Since(s.slice[0].t))
		s.mutex.RUnlock()

		<-timer

		s.mutex.Lock()
		delete(s.set, s.slice[0].v)
		s.slice = s.slice[1:]

		if len(s.slice) == 0 {
			s.mutex.Unlock()
			return
		}
		s.mutex.Unlock()
	}
}
