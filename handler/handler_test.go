package handler

import (
	"errors"
	"net/url"

	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/strategy"
)

// fakeStrategy will return ok values and capture any args called with
type fakeStrategy struct {
	match        *url.URL
	expectedLink string
	form         url.Values
}

func (fakeStrategy) Name() string {
	return "fake"
}

func (s *fakeStrategy) Match(me *url.URL) bool {
	s.match = me
	return true
}

func (s *fakeStrategy) Redirect(expectedLink string) (redirectURL string, err error) {
	s.expectedLink = expectedLink
	return "https://example.com/redirect", nil
}

func (s *fakeStrategy) Callback(form url.Values) (string, error) {
	s.form = form
	return "me", nil
}

// falseStrategy is a strategy that can never be matched
type falseStrategy struct{}

func (falseStrategy) Name() string {
	return "false"
}

func (falseStrategy) Match(me *url.URL) bool {
	return false
}

func (falseStrategy) Redirect(expectedLink string) (redirectURL string, err error) {
	return "https://example.com/redirect", nil
}

func (falseStrategy) Callback(form url.Values) (string, error) {
	return "me", nil
}

// unauthorizedStrategy is a strategy for scenarios where the user is reported as unauthorized
type unauthorizedStrategy struct{}

func (unauthorizedStrategy) Name() string           { return "unauthorized" }
func (unauthorizedStrategy) Match(me *url.URL) bool { return true }
func (unauthorizedStrategy) Redirect(expectedLink string) (string, error) {
	return "https://example.com/redirect", nil
}
func (unauthorizedStrategy) Callback(form url.Values) (string, error) {
	return "", strategy.ErrUnauthorized
}

// errorStrategy is a strategy for scenarios where the provider errors
type errorStrategy struct{}

func (errorStrategy) Name() string           { return "error" }
func (errorStrategy) Match(me *url.URL) bool { return true }
func (errorStrategy) Redirect(expectedLink string) (string, error) {
	return "https://example.com/redirect", nil
}
func (errorStrategy) Callback(form url.Values) (string, error) {
	return "", errors.New("/shrug")
}

// fakeSessionStore can be used as data.SessionStore
type fakeSessionStore struct {
	Session data.Session
}

func (s *fakeSessionStore) Save(session *data.Session) {}

func (s *fakeSessionStore) Update(session data.Session) {}

func (s *fakeSessionStore) Get(me string) (data.Session, bool) {
	if me == s.Session.Me {
		return s.Session, true
	}

	return data.Session{}, false
}

func (s *fakeSessionStore) GetByCode(code string) (data.Session, bool) {
	if code == s.Session.Code {
		return s.Session, true
	}

	return data.Session{}, false
}

func (s *fakeSessionStore) GetByToken(token string) (data.Session, bool) {
	if token == s.Session.Token {
		return s.Session, true
	}

	return data.Session{}, false
}

func (s *fakeSessionStore) RevokeByToken(token string) {
	if token == s.Session.Token {
		s.Session = data.Session{}
	}
}
