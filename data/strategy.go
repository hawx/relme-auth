package data

import (
	"sync"
	"time"

	"hawx.me/code/relme-auth/random"
)

type StrategyStore struct {
	expiry     int64
	mu         sync.Mutex
	inProgress map[string]*expiringItem
}

type expiringItem struct {
	createdAt int64
	value     string
}

func Strategy(name string) (*StrategyStore, error) {
	return &StrategyStore{
		inProgress: map[string]*expiringItem{},
		expiry:     60,
	}, nil
}

func (s *StrategyStore) Insert(link string) (state string, err error) {
	state, err = random.String(64)
	if err != nil {
		return
	}

	return state, s.Set(state, link)
}

func (s *StrategyStore) Set(key, value string) error {
	s.mu.Lock()
	s.inProgress[key] = &expiringItem{createdAt: time.Now().Unix(), value: value}
	s.mu.Unlock()

	return nil
}

func (s *StrategyStore) Claim(state string) (link string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.inProgress[state]
	if !ok {
		return "", false
	}

	delete(s.inProgress, state)

	if time.Now().Unix()-item.createdAt > s.expiry {
		return "", false
	}

	return item.value, true
}
