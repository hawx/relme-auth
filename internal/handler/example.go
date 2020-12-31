package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"hawx.me/code/relme-auth/internal/config"
	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/random"
)

type exampleStore interface {
	CreateToken(data.Token) error
	Tokens(string) ([]data.Token, error)
	RevokeRow(me, rowID string) error
	Forget(string) error
}

// Example implements a basic site using the authentication flow provided by
// this package.
func Example(baseURL string, conf config.Config, store sessions.Store, tokenStore exampleStore, templates tmpl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		var me string
		var tokens []data.Token

		meValue, ok := session.Values["me"]
		if ok {
			me, ok = meValue.(string)
		}
		if ok {
			tokens, _ = tokenStore.Tokens(me)
		}

		state, _ := random.String(64)
		session.Values["state"] = state
		codeVerifier, _ := random.String(32)
		session.Values["verifier"] = codeVerifier
		if err := session.Save(r, w); err != nil {
			log.Println("handler/example could not save session:", err)
		}

		hashedVerifier := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := strings.TrimRight(base64.URLEncoding.EncodeToString(hashedVerifier[:]), "=")

		if err := templates.ExecuteTemplate(w, "welcome.gotmpl", welcomeCtx{
			ThisURI:       baseURL,
			State:         state,
			CodeChallenge: codeChallenge,
			Me:            me,
			LoggedIn:      ok,
			HasFlickr:     conf.Flickr != nil,
			HasGitHub:     conf.GitHub != nil,
			HasTwitter:    conf.Twitter != nil,
			Tokens:        tokens,
		}); err != nil {
			log.Println("handler/example failed to write template:", err)
		}
	}
}

// ExampleCallback implements the authentication callback for Example. It
// verifies the code, then sets the value of "me" in a session cookie.
func ExampleCallback(baseURL string, store sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		code := r.FormValue("code")
		state := r.FormValue("state")

		if state != session.Values["state"] {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		redirectURL := baseURL + "/callback"
		finalURL := baseURL

		if r.FormValue("r") == "privacy" {
			redirectURL += "?r=privacy"
			finalURL = baseURL + "/privacy"
		}

		resp, err := http.PostForm(baseURL+"/auth", url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"client_id":     {baseURL + "/"},
			"redirect_uri":  {redirectURL},
			"code_verifier": {session.Values["verifier"].(string)},
		})
		if err != nil || resp.StatusCode != 200 {
			io.Copy(w, resp.Body)
			http.Error(w, "could not authenticate", http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()

		var v exampleResponse
		if err = json.NewDecoder(resp.Body).Decode(&v); err != nil {
			http.Error(w, "response had a weird body", http.StatusInternalServerError)
			return
		}

		session.Values["me"] = v.Me
		if err := session.Save(r, w); err != nil {
			log.Println("handler/example could not save session:", err)
		}

		http.Redirect(w, r, finalURL, http.StatusFound)
	}
}

// ExampleSignOut removes the value of "me" from the session cookie.
func ExampleSignOut(baseURL string, store sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		delete(session.Values, "me")
		if err := session.Save(r, w); err != nil {
			log.Println("handler/example could not save session:", err)
		}

		http.Redirect(w, r, baseURL, http.StatusFound)
	}
}

// ExampleRevoke removes the token for the client_id.
func ExampleRevoke(baseURL string, store sessions.Store, tokenStore exampleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		if r.FormValue("state") != session.Values["state"] {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		me, ok := session.Values["me"].(string)
		if !ok {
			return
		}

		rowID := r.FormValue("id")

		if err := tokenStore.RevokeRow(me, rowID); err != nil {
			log.Println("handler/example failed to revoke token:", err)
		}

		http.Redirect(w, r, baseURL, http.StatusFound)
	}
}

func ExampleGenerate(
	baseURL string,
	store sessions.Store,
	generator func() (string, error),
	tokenStore exampleStore,
	templates tmpl,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		if r.FormValue("state") != session.Values["state"] {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		me, ok := session.Values["me"].(string)
		if !ok {
			return
		}

		tokenString, err := generator()
		if err != nil {
			log.Println("handler/token could not generate token:", err)
			return
		}

		if err := tokenStore.CreateToken(data.Token{
			Token:     tokenString,
			Me:        me,
			ClientID:  r.FormValue("client_id"),
			Scope:     r.FormValue("scope"),
			CreatedAt: time.Now(),
		}); err != nil {
			log.Println("handler/example failed to create token:", err)
		}

		if err := templates.ExecuteTemplate(w, "generate.gotmpl", struct {
			ThisURI  string
			Me       string
			ClientID string
			Token    string
		}{
			ThisURI:  baseURL,
			Me:       me,
			ClientID: r.FormValue("client_id"),
			Token:    tokenString,
		}); err != nil {
			log.Println("handler/example failed to write template:", err)
		}
	}
}

func ExamplePrivacy(baseURL string, store sessions.Store, templates tmpl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		var me string
		meValue, ok := session.Values["me"]
		if ok {
			me, ok = meValue.(string)
		}

		state, _ := random.String(64)
		session.Values["state"] = state
		if err := session.Save(r, w); err != nil {
			log.Println("handler/example could not save session:", err)
		}

		if err := templates.ExecuteTemplate(w, "privacy.gotmpl", privacyCtx{
			ThisURI:  baseURL,
			Me:       me,
			LoggedIn: ok,
			State:    state,
		}); err != nil {
			log.Println("handler/example failed to write template:", err)
		}
	}
}

func ExampleForget(baseURL string, store sessions.Store, tokenStore exampleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		var me string
		meValue, ok := session.Values["me"]
		if ok {
			me, ok = meValue.(string)
		}
		if ok {
			if err := tokenStore.Forget(me); err != nil {
				log.Println("handler/example failed to forget:", err)
				http.Error(w, "failed to forget", http.StatusInternalServerError)
				return
			}

			delete(session.Values, "me")
			if err := session.Save(r, w); err != nil {
				log.Println("handler/example could not save session:", err)
			}
		}

		http.Redirect(w, r, baseURL, http.StatusFound)
	}
}

type exampleResponse struct {
	Me string `json:"me"`
}

type welcomeCtx struct {
	ThisURI       string
	State         string
	CodeChallenge string
	Me            string
	LoggedIn      bool
	HasFlickr     bool
	HasGitHub     bool
	HasTwitter    bool
	Tokens        []data.Token
}

type privacyCtx struct {
	ThisURI  string
	State    string
	Me       string
	LoggedIn bool
}
