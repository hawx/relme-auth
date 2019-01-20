package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
)

// Example implmenets a basic site using the authentication flow provided by
// this package.
func Example(baseURL string) http.Handler {
	mux := http.NewServeMux()
	store := sessions.NewCookieStore([]byte("something-very-secret"))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
  <head>
    <title>Example site</title>
    <style>
      body {
        font: 16px/1.3 sans-serif;
        margin: 2rem;
      }

      h1 {
        font-size: 1.6rem;
      }

      label {
        display: block;
      }
    </style>
  </head>
  <body class="h-x-app">
    <h1>Sign-in to <span class="p-name">Example site</span></h1>

    <form action="/auth" method="get">
      <label for="web_address">Web Address</label>
      <input id="web_address" type="text" name="me" placeholder="https://yourdomain.com" />
      <p><button type="submit">Sign In</button></p>
      <input type="hidden" name="client_id" value="%[1]s/" />
      <input type="hidden" name="redirect_uri" value="%[1]s/callback" />
    </form>
  </body>
</html>
`, baseURL)
	})

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")

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

		session, err := store.Get(r, "example-session")
		if err != nil {
			http.Error(w, "can't set cookies...", http.StatusInternalServerError)
			return
		}

		session.Values["me"] = v.Me
		session.Save(r, w)

		http.Redirect(w, r, baseURL+"/success", http.StatusFound)
	})

	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "example-session")
		if err != nil {
			http.Error(w, "can't get cookies...", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
  <head>
    <title>Example site</title>
    <style>
      body {
        font: 16px/1.3 sans-serif;
        margin: 2rem;
      }
    </style>
  </head>
  <body>
    <h1>Example site</h1>
    <p>You are signed-in as <a href="%[1]s">%[1]s</a>.</p>
  </body>
</html>
`, session.Values["me"])
	})

	mux.HandleFunc("/failure", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<body>
  <p>Something went wrong when trying to sign-in.</p>
</body>
</html>
`)
	})

	return context.ClearHandler(mux)
}

type exampleResponse struct {
	Me string `json:"me"`
}
