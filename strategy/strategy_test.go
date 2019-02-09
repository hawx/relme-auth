package strategy

import (
	"errors"
	"sync"
)

type fakeStore struct {
	mu         sync.Mutex
	inProgress map[string]interface{}
}

func (s *fakeStore) Insert(link interface{}) (state string, err error) {
	if s.inProgress == nil {
		s.inProgress = map[string]interface{}{}
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

func (s *fakeStore) Set(key string, value interface{}) error {
	if s.inProgress == nil {
		s.inProgress = map[string]interface{}{}
	}

	s.mu.Lock()
	s.inProgress[key] = value
	s.mu.Unlock()

	return nil
}

func (s *fakeStore) Claim(state string) (link interface{}, ok bool) {
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

type oneStore struct {
	State string
	Link  interface{}
}

func (s *oneStore) Insert(link interface{}) (state string, err error) {
	s.Link = link

	return s.State, nil
}

func (s *oneStore) Set(key string, value interface{}) error {
	return errors.New("not used")
}

func (s *oneStore) Claim(state string) (link interface{}, ok bool) {
	if state != s.State {
		return "", false
	}

	return s.Link, true
}
