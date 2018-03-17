package state

import (
	"sync"
	"time"
)

type authStore struct {
	mu         sync.Mutex
	inProgress map[string]string
	sessions   []*Session
}

type StrategyStore interface {
	Insert(link string) (state string, err error)
	Set(key, value string) error
	Claim(state string) (link string, ok bool)
}

type Store interface {
	StrategyStore

	Save(*Session)
	Get(string) (Session, bool)
	GetByCode(string) (Session, bool)
}

type Session struct {
	Me          string
	Code        string
	ClientID    string
	RedirectURI string
	CreatedAt   time.Time
}

func (s Session) Expired() bool {
	return time.Now().Add(-60 * time.Second).After(s.CreatedAt)
}

func NewStore() *authStore {
	return &authStore{
		inProgress: map[string]string{},
		sessions:   []*Session{},
	}
}

func (store *authStore) Save(session *Session) {
	session.CreatedAt = time.Now()
	session.Code, _ = randomString(16)
	store.sessions = append(store.sessions, session)
}

func (store *authStore) Get(me string) (session Session, ok bool) {
	for _, session := range store.sessions {
		if session.Me == me {
			return *session, true
		}
	}

	return
}

func (store *authStore) GetByCode(code string) (session Session, ok bool) {
	for _, session := range store.sessions {
		if session.Code == code {
			return *session, true
		}
	}

	return
}

func (store *authStore) Insert(link string) (state string, err error) {
	state, err = randomString(64)
	if err != nil {
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	store.inProgress[state] = link

	return
}

func (store *authStore) Set(key, value string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.inProgress[key] = value
	return nil
}

func (store *authStore) Claim(state string) (link string, ok bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	link, ok = store.inProgress[state]
	if ok {
		delete(store.inProgress, state)
	}

	return
}
