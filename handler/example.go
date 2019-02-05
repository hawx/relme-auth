package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	"hawx.me/code/relme-auth/config"
	"hawx.me/code/relme-auth/data"
)

// Example implements a basic site using the authentication flow provided by
// this package.
func Example(baseURL string, conf config.Config, store sessions.Store, templates *template.Template) http.HandlerFunc {
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
			state, _ = data.RandomString(64)
			session.Values["state"] = state
			session.Save(r, w)
		}

		templates.ExecuteTemplate(w, "welcome.gotmpl", welcomeCtx{
			ThisURI:    baseURL,
			State:      state,
			Me:         me,
			LoggedIn:   ok,
			HasFlickr:  conf.Flickr != nil,
			HasGitHub:  conf.GitHub != nil,
			HasTwitter: conf.Twitter != nil,
		})
	}
}

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
		session.Save(r, w)

		http.Redirect(w, r, baseURL, http.StatusFound)
	}
}

func ExampleSignOut(baseURL string, store sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "example-session")
		if err != nil {
			http.Error(w, "can't get cookies...", http.StatusInternalServerError)
			return
		}

		delete(session.Values, "me")
		session.Save(r, w)

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
