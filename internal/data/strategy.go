package data

import (
	"sync"
	"time"

	"hawx.me/code/relme-auth/internal/random"
)

type StrategyStore struct {
	expiry     int64
	mu         sync.Mutex
	inProgress map[string]*expiringItem
}

type expiringItem struct {
	createdAt int64
	value     interface{}
}

func Strategy(name string) (*StrategyStore, error) {
	return &StrategyStore{
		inProgress: map[string]*expiringItem{},
		expiry:     60,
	}, nil
}

func (s *StrategyStore) Insert(value interface{}) (state string, err error) {
	state, err = random.String(64)
	if err != nil {
		return
	}

	return state, s.Set(state, value)
}

func (s *StrategyStore) Set(key string, value interface{}) error {
	s.mu.Lock()
	s.inProgress[key] = &expiringItem{createdAt: time.Now().Unix(), value: value}
	s.mu.Unlock()

	return nil
}

func (s *StrategyStore) Claim(key string) (value interface{}, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.inProgress[key]
	if !ok {
		return "", false
	}

	delete(s.inProgress, key)

	if time.Now().Unix()-item.createdAt > s.expiry {
		return "", false
	}

	return item.value, true
}
