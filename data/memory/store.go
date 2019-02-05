package memory

import (
	"errors"
	"sync"
	"time"

	"hawx.me/code/relme-auth/data"
)

type authStore struct {
	mu         sync.Mutex
	inProgress map[string]string
	sessions   []*data.Session
	profiles   map[string]data.Profile
	clients    map[string]data.Client
}

func New() data.Database {
	return &authStore{
		inProgress: map[string]string{},
		sessions:   []*data.Session{},
		profiles:   map[string]data.Profile{},
		clients:    map[string]data.Client{},
	}
}

func (s *authStore) Close() error {
	return nil
}

func (s *authStore) CacheProfile(profile data.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profiles[profile.Me] = profile
	return nil
}

func (s *authStore) CacheClient(client data.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client.ID] = client
	return nil
}

func (s *authStore) GetProfile(me string) (data.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.profiles[me]
	if !ok {
		return profile, errors.New("no such profile")
	}

	return profile, nil
}

func (s *authStore) GetClient(clientID string) (data.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[clientID]
	if !ok {
		return client, errors.New("no such client")
	}

	return client, nil
}

func (s *authStore) Save(session *data.Session) {
	session.CreatedAt = time.Now()
	session.Code, _ = data.RandomString(16)
	session.Token, _ = data.RandomString(32)
	s.sessions = append(s.sessions, session)
}

func (s *authStore) Update(session data.Session) {
	for _, found := range s.sessions {
		if found.Me == session.Me {
			*found = session
			return
		}
	}
}

func (s *authStore) Get(me string) (session data.Session, ok bool) {
	for _, session := range s.sessions {
		if session.Me == me {
			return *session, true
		}
	}

	return
}

func (s *authStore) GetByCode(code string) (session data.Session, ok bool) {
	for _, session := range s.sessions {
		if session.Code == code {
			return *session, true
		}
	}

	return
}

func (s *authStore) GetByToken(token string) (session data.Session, ok bool) {
	for _, session := range s.sessions {
		if session.Token == token {
			return *session, true
		}
	}

	return
}

func (s *authStore) RevokeByToken(token string) {
	var idx int
	for i, session := range s.sessions {
		if session.Token == token {
			idx = i
			break
		}
	}

	s.sessions = append(s.sessions[:idx], s.sessions[idx+1:]...)

	return
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
