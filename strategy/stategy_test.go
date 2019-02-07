package strategy

import (
	"sync"
)

type fakeStore struct {
	mu         sync.Mutex
	inProgress map[string]string
}

func (s *fakeStore) Insert(link string) (state string, err error) {
	if s.inProgress == nil {
		s.inProgress = map[string]string{}
	}

	state, err = randomString(64)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.inProgress[state] = link
	s.mu.Unlock()

	return
}

func (s *fakeStore) Set(key, value string) error {
	if s.inProgress == nil {
		s.inProgress = map[string]string{}
	}

	s.mu.Lock()
	s.inProgress[key] = value
	s.mu.Unlock()

	return nil
}

func (s *fakeStore) Claim(state string) (link string, ok bool) {
	if s.inProgress == nil {
		return "", false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	link, ok = s.inProgress[state]
	if ok {
		delete(s.inProgress, state)
	}

	return
}
