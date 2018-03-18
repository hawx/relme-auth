package memory

import (
	"crypto/rand"
	"sync"
	"time"

	"hawx.me/code/relme-auth/store"
)

type authStore struct {
	mu         sync.Mutex
	inProgress map[string]string
	sessions   []*store.Session
}

func NewStore() *authStore {
	return &authStore{
		inProgress: map[string]string{},
		sessions:   []*store.Session{},
	}
}

func (s *authStore) Save(session *store.Session) {
	session.CreatedAt = time.Now()
	session.Code, _ = randomString(16)
	s.sessions = append(s.sessions, session)
}

func (s *authStore) Get(me string) (session store.Session, ok bool) {
	for _, session := range s.sessions {
		if session.Me == me {
			return *session, true
		}
	}

	return
}

func (s *authStore) GetByCode(code string) (session store.Session, ok bool) {
	for _, session := range s.sessions {
		if session.Code == code {
			return *session, true
		}
	}

	return
}

func (s *authStore) Insert(link string) (state string, err error) {
	state, err = randomString(64)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.inProgress[state] = link
	s.mu.Unlock()

	return
}

func (s *authStore) Set(key, value string) error {
	s.mu.Lock()
	s.inProgress[key] = value
	s.mu.Unlock()

	return nil
}

func (s *authStore) Claim(state string) (link string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	link, ok = s.inProgress[state]
	if ok {
		delete(s.inProgress, state)
	}

	return
}

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"

func randomString(n int) (string, error) {
	bytes, err := randomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

func randomBytes(length int) (b []byte, err error) {
	b = make([]byte, length)
	_, err = rand.Read(b)
	return
}
