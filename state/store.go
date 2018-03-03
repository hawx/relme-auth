package state

import "sync"

type authStore struct {
	mu         sync.Mutex
	inProgress map[string]string
}

type Store interface {
	Insert(link string) (state string, err error)
	Set(key, value string) error
	Claim(state string) (link string, ok bool)
}

func NewStore() *authStore {
	return &authStore{
		inProgress: map[string]string{},
	}
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
