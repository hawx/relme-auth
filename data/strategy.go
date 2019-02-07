package data

import (
	"sync"
)

type StrategyStore struct {
	mu         sync.Mutex
	inProgress map[string]string
}

func Strategy(name string) (*StrategyStore, error) {
	return &StrategyStore{inProgress: map[string]string{}}, nil
}

func (s *StrategyStore) Insert(link string) (state string, err error) {
	state, err = RandomString(64)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.inProgress[state] = link
	s.mu.Unlock()

	return
}

func (s *StrategyStore) Set(key, value string) error {
	s.mu.Lock()
	s.inProgress[key] = value
	s.mu.Unlock()

	return nil
}

func (s *StrategyStore) Claim(state string) (link string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	link, ok = s.inProgress[state]
	if ok {
		delete(s.inProgress, state)
	}

	return
}
