package memory

import (
	"sync"

	"hawx.me/code/relme-auth/data"
)

type authStore struct {
}

// New returns an empty in memory database.
func New() data.Database {
	return &authStore{}
}

func (s *authStore) Close() error {
	return nil
}

type strategyStore struct {
	mu         sync.Mutex
	inProgress map[string]string
}

func (s *authStore) Strategy(name string) (data.StrategyStore, error) {
	return &strategyStore{inProgress: map[string]string{}}, nil
}

func (s *strategyStore) Insert(link string) (state string, err error) {
	state, err = data.RandomString(64)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.inProgress[state] = link
	s.mu.Unlock()

	return
}

func (s *strategyStore) Set(key, value string) error {
	s.mu.Lock()
	s.inProgress[key] = value
	s.mu.Unlock()

	return nil
}

func (s *strategyStore) Claim(state string) (link string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	link, ok = s.inProgress[state]
	if ok {
		delete(s.inProgress, state)
	}

	return
}
