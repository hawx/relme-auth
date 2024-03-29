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

	"github.com/gorilla/sessions"
	"hawx.me/code/relme-auth/internal/config"
	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/random"
)

type ExampleDB interface {
	CreateToken(data.Token) error
	Tokens(string) ([]data.Token, error)
	RevokeToken(string) error
	Forget(string) error
}

// Example implements a basic site using the authentication flow provided by
// this package.
func Example(baseURL string, conf config.Config, store sessions.Store, tokenStore ExampleDB, welcomeTemplate, accountTemplate tmpl) http.HandlerFunc {
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
			state, _ := random.String(64)
			session.Values["state"] = state

			if err := session.Save(r, w); err != nil {
				log.Println("handler/example could not save session:", err)
			}

			if err := accountTemplate.ExecuteTemplate(w, "page", accountCtx{
				Me:     me,
				State:  state,
				Tokens: tokens,
			}); err != nil {
				log.Println("handler/example failed to write template:", err)
			}

			return
		}

		if err := welcomeTemplate.ExecuteTemplate(w, "page", welcomeCtx{
			ThisURI:    baseURL,
			Me:         me,
			LoggedIn:   ok,
			HasFlickr:  conf.Flickr != nil,
			HasGitHub:  conf.GitHub != nil,
			HasTwitter: conf.Twitter != nil,
			Tokens:     tokens,
		}); err != nil {
			log.Println("handler/example failed to write template:", err)
		}
	}
}

func ExampleSignIn(baseURL string, store sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		state, _ := random.String(64)
		session.Values["state"] = state
		codeVerifier, _ := random.String(32)
		session.Values["verifier"] = codeVerifier
		if err := session.Save(r, w); err != nil {
			log.Println("handler/example could not save session:", err)
		}

		hashedVerifier := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := strings.TrimRight(base64.URLEncoding.EncodeToString(hashedVerifier[:]), "=")
		form := url.Values{
			"response_type":         {"code"},
			"client_id":             {baseURL},
			"redirect_uri":          {baseURL + "/redirect"},
			"state":                 {state},
			"code_challenge":        {codeChallenge},
			"code_challenge_method": {"S256"},
			"me":                    {r.FormValue("me")},
		}

		redirectURL := baseURL + "/auth?" + form.Encode()

		http.Redirect(w, r, redirectURL, http.StatusFound)
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

		redirectURL := baseURL + "/redirect"

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

		http.Redirect(w, r, "/", http.StatusFound)
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
func ExampleRevoke(baseURL string, store sessions.Store, tokenStore ExampleDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "example-session")

		if r.FormValue("state") != session.Values["state"] {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		rowID := r.FormValue("id")

		if err := tokenStore.RevokeToken(rowID); err != nil {
			log.Println("handler/example failed to revoke token:", err)
		}

		http.Redirect(w, r, baseURL, http.StatusFound)
	}
}

func ExampleGenerate(
	baseURL string,
	store sessions.Store,
	generator func(int) (string, error),
	tokenStore ExampleDB,
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

		token, tokenString, err := data.NewToken(generator, data.Code{
			Me:       me,
			ClientID: r.FormValue("client_id"),
			Scope:    r.FormValue("scope"),
		})
		if err != nil {
			log.Println("handler/token could not generate token:", err)
			return
		}

		if err := tokenStore.CreateToken(token); err != nil {
			log.Println("handler/example failed to create token:", err)
		}

		if err := templates.ExecuteTemplate(w, "page", struct {
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

func ExamplePrivacy(templates tmpl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, "page", nil); err != nil {
			log.Println("handler/example failed to write template:", err)
		}
	}
}

func ExampleForget(baseURL string, store sessions.Store, tokenStore ExampleDB) http.HandlerFunc {
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

type accountCtx struct {
	State  string
	Me     string
	Tokens []data.Token
}
