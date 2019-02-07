package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	"hawx.me/code/relme-auth/config"
	"hawx.me/code/relme-auth/random"
)

// Example implements a basic site using the authentication flow provided by
// this package.
func Example(baseURL string, conf config.Config, store sessions.Store, templates tmpl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "example-session")
		if err != nil {
			http.Error(w, "can't get cookies...", http.StatusInternalServerError)
			return
		}

		var me string
		meValue, ok := session.Values["me"]
		if ok {
			me, ok = meValue.(string)
		}

		var state string
		if !ok {
			state, _ = random.String(64)
			session.Values["state"] = state
			if err := session.Save(r, w); err != nil {
				log.Println("handler/example could not save session:", err)
			}
		}

		if err := templates.ExecuteTemplate(w, "welcome.gotmpl", welcomeCtx{
			ThisURI:    baseURL,
			State:      state,
			Me:         me,
			LoggedIn:   ok,
			HasFlickr:  conf.Flickr != nil,
			HasGitHub:  conf.GitHub != nil,
			HasTwitter: conf.Twitter != nil,
		}); err != nil {
			log.Println("handler/example failed to write template:", err)
		}
	}
}

// ExampleCallback implements the authentication callback for Example. It
// verifies the code, then sets the value of "me" in a session cookie.
func ExampleCallback(baseURL string, store sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "example-session")
		if err != nil {
			http.Error(w, "can't get cookies...", http.StatusInternalServerError)
			return
		}

		code := r.FormValue("code")
		state := r.FormValue("state")

		if state != session.Values["state"] {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		resp, err := http.PostForm(baseURL+"/auth", url.Values{
			"code":         {code},
			"client_id":    {baseURL + "/"},
			"redirect_uri": {baseURL + "/callback"},
		})
		if err != nil || resp.StatusCode != 200 {
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

		http.Redirect(w, r, baseURL, http.StatusFound)
	}
}

// ExampleSignOut removes the value of "me" from the session cookie.
func ExampleSignOut(baseURL string, store sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "example-session")
		if err != nil {
			http.Error(w, "can't get cookies...", http.StatusInternalServerError)
			return
		}

		delete(session.Values, "me")
		if err := session.Save(r, w); err != nil {
			log.Println("handler/example could not save session:", err)
		}

		http.Redirect(w, r, baseURL, http.StatusFound)
	}
}

type exampleResponse struct {
	Me string `json:"me"`
}

type welcomeCtx struct {
	ThisURI    string
	State      string
	Me         string
	LoggedIn   bool
	HasFlickr  bool
	HasGitHub  bool
	HasTwitter bool
}
