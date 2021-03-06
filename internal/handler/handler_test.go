package handler

import (
	"errors"
	"net/url"

	"hawx.me/code/relme-auth/internal/strategy"
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

func (s *fakeStrategy) Redirect(expectedLink, _ string) (redirectURL string, err error) {
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

func (falseStrategy) Redirect(expectedLink, _ string) (redirectURL string, err error) {
	return "https://example.com/redirect", nil
}

func (falseStrategy) Callback(form url.Values) (string, error) {
	return "me", nil
}

// unauthorizedStrategy is a strategy for scenarios where the user is reported as unauthorized
type unauthorizedStrategy struct{}

func (unauthorizedStrategy) Name() string           { return "unauthorized" }
func (unauthorizedStrategy) Match(me *url.URL) bool { return true }
func (unauthorizedStrategy) Redirect(expectedLink, _ string) (string, error) {
	return "https://example.com/redirect", nil
}
func (unauthorizedStrategy) Callback(form url.Values) (string, error) {
	return "", strategy.ErrUnauthorized
}

// errorStrategy is a strategy for scenarios where the provider errors
type errorStrategy struct{}

func (errorStrategy) Name() string           { return "error" }
func (errorStrategy) Match(me *url.URL) bool { return true }
func (errorStrategy) Redirect(expectedLink, _ string) (string, error) {
	return "https://example.com/redirect", nil
}
func (errorStrategy) Callback(form url.Values) (string, error) {
	return "", errors.New("/shrug")
}
