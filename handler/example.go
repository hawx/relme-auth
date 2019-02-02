package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"hawx.me/code/relme-auth/config"
	"hawx.me/code/relme-auth/data"
)

// Example implmenets a basic site using the authentication flow provided by
// this package.
func Example(baseURL string, conf config.Config) http.Handler {
	mux := http.NewServeMux()
	store := sessions.NewCookieStore([]byte("something-very-secret"))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

		welcomeTmpl.Execute(w, welcomeCtx{
			ThisURI:    baseURL,
			State:      state,
			Me:         me,
			LoggedIn:   ok,
			HasFlickr:  conf.Flickr != nil,
			HasGitHub:  conf.GitHub != nil,
			HasTwitter: conf.Twitter != nil,
		})
	})

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
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
	})

	mux.HandleFunc("/sign-out", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "example-session")
		if err != nil {
			http.Error(w, "can't get cookies...", http.StatusInternalServerError)
			return
		}

		delete(session.Values, "me")
		session.Save(r, w)

		http.Redirect(w, r, baseURL, http.StatusFound)
	})

	return context.ClearHandler(mux)
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

var welcomeTmpl = template.Must(template.New("welcome").Parse(welcomePage))

const welcomePage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>relme-auth</title>
    <style>
      /* http://meyerweb.com/eric/tools/css/reset/
         v2.0 | 20110126
         License: none (public domain)
       */

      html, body, div, span, applet, object, iframe,
      h1, h2, h3, h4, h5, h6, p, blockquote, pre,
      a, abbr, acronym, address, big, cite, code,
      del, dfn, em, img, ins, kbd, q, s, samp,
      small, strike, strong, sub, sup, tt, var,
      b, u, i, center,
      dl, dt, dd, ol, ul, li,
      fieldset, form, label, legend,
      table, caption, tbody, tfoot, thead, tr, th, td,
      article, aside, canvas, details, embed,
      figure, figcaption, footer, header, hgroup,
      menu, nav, output, ruby, section, summary,
      time, mark, audio, video {
	      margin: 0;
	      padding: 0;
	      border: 0;
	      font-size: 100%;
	      font: inherit;
	      vertical-align: baseline;
      }
      /* HTML5 display-role reset for older browsers */
      article, aside, details, figcaption, figure,
      footer, header, hgroup, menu, nav, section {
	      display: block;
      }
      body {
	      line-height: 1;
      }
      ol, ul {
	      list-style: none;
      }
      blockquote, q {
	      quotes: none;
      }
      blockquote:before, blockquote:after,
      q:before, q:after {
	      content: '';
	      content: none;
      }
      table {
	      border-collapse: collapse;
	      border-spacing: 0;
      }

      /* start */

      body {
        font: 18px/1.3 sans-serif;
      }

      @media screen and (min-width: 100rem) {
        @supports (display: grid) {
          body {
            display: grid;
            grid-template-areas: "header header" "users developers" "footer footer";
          }
        }
      }

      #users {
        grid-area: users;
      }
      #developers {
        grid-area: developers;
      }
      footer {
        grid-area: footer;
        font-size: .9rem;
        text-align: center;
        padding: 2rem 0 3rem;
      }

      header {
        grid-area: header;
        height: 70vh;
        display: flex;
        flex-direction: column;
        justify-content: space-around;
        text-align: center;
        background: rgb(240, 240, 240);
        border-bottom: 1px solid rgb(210, 210, 210)
      }

      header h1 {
        font-size: 2rem;
        font-weight: bold;
        font-family: monospace;
      }

      header h2 {
        font-style: italic;
        margin-bottom: 2rem;
      }

      header p {
        margin: 1rem;
      }

      .field {
        display: flex;
      }

      .field input {
        margin-right: 5px;
      }

      .container {
        max-width: 50rem;
        margin: 0 auto;
        padding: 2rem 4rem;
      }

      section h1 {
        font-size: 1.6rem;
        margin: 1rem -1rem;
      }
      section h2 {
        font-size: 1.2rem;
        margin: 1rem -1rem;
      }

      pre {
        padding: 1rem;
        margin: 1rem 0;
        background: rgb(240, 240, 240);
        overflow-x: scroll;
      }

      code {
        font: .9rem/1.3 monospace;
      }
      strong { font-weight: bold; }
      dl {
        margin-top: 1rem;
      }
      dd {
        margin-bottom: 1rem;
      }
    </style>
  </head>
  <body>
    <header class="h-app">
      <div class="container">
        <h1 class="p-name">relme-auth</h1>
        <h2>Sign in with your domain</h2>

        {{ if .LoggedIn }}
        <p>You are logged in as <strong>{{ .Me }}</strong>.</p>
        <form action="/sign-out" method="post">
          <button type="submit">Sign Out</button>
        </form>
        {{ end }}{{ if not .LoggedIn }}<p>Try signing in to this site:</p>

        <form action="/auth" method="get">
          <div class="field">
            <input type="url" name="me" placeholder="https://example.com" />
            <button type="submit">Sign In</button>
          </div>
          <input type="hidden" name="client_id" value="{{ .ThisURI }}/" />
          <input type="hidden" name="redirect_uri" value="{{ .ThisURI }}/callback" />
          <input type="hidden" name="state" value="{{ .State }}" />
        </form>{{ end }}
      </div>
    </header>

    <section id="users">
      <div class="container">
        <h1>For users</h1>

        <p>You can log in to this site without creating a new account! Instead make sure one (or more) of
          the methods below is setup.</p>

        {{ if .HasFlickr }}<h2 id="flickr">Flickr</h2>
        <ol>
          <li>To authenticate with your Flickr account add a link to your profile on your homepage.
            <pre><code>&lt;a rel="me" href="https://www.flickr.com/people/YOU"&gt;Flickr&lt;/a&gt;</code></pre>
            Or if you don't want the link to be visible.
            <pre><code>&lt;link rel="me" href="https://www.flickr.com/people/YOU" /&gt;</code></pre>
          </li>
          <li>Make sure your Flickr profile has a link back to your homepage.</li>
        </ol>{{ end }}

        {{ if .HasGitHub }}<h2 id="github">GitHub</h2>
        <ol>
          <li>To authenticate with your GitHub account add a link to your profile on your homepage.
            <pre><code>&lt;a rel="me" href="https://github.com/YOU"&gt;GitHub&lt;/a&gt;</code></pre>
            Or if you don't want the link to be visible.
            <pre><code>&lt;link rel="me" href="https://github.com/YOU" /&gt;</code></pre>
          </li>
          <li>Make sure your GitHub profile has a link back to your homepage.</li>
        </ol>{{ end }}

        {{ if .HasTwitter }}<h2 id="twitter">Twitter</h2>
        <ol>
          <li>To authenticate with your Twitter account add a link to your profile on your homepage.
            <pre><code>&lt;a rel="me" href="https://twitter.com/YOU"&gt;Twitter&lt;/a&gt;</code></pre>
            Or if you don't want the link to be visible.
            <pre><code>&lt;link rel="me" href="https://twitter.com/YOU" /&gt;</code></pre>
          </li>
          <li>Make sure your Twitter profile has a link back to your homepage.</li>
        </ol>{{ end }}
      </div>
    </section>

    <section id="developers">
      <div class="container">
        <h1>For developers</h1>

        <p>It is possible to use this site to provide login for your users.</p>

        <h2>Redirect a user to relme-auth</h2>
        <p>The first step is to send a user to relme-auth so they can choose how
          to authenticate. This is a simple redirect to <code>{{ .ThisURI }}/auth</code> with
          a few query parameters:</p>
        <dl>
          <dt><code>me=</code></dt>
          <dd>The web address of the user who is logging in.</dd>
          <dt><code>client_id=</code></dt>
          <dd>The URL to the site they are logging in to. This <em>should</em> be marked up with <a href="https://www.w3.org/TR/indieauth/#application-information"><code>h-app</code></a> to provide a name to
            display. You are also able to whitelist a <a href="https://www.w3.org/TR/indieauth/#redirect-url"><code>redirect_uri</code></a> if it is not hosted at the same domain.</dd>
          <dt><code>redirect_uri=</code></dt>
          <dd>Where to send the user after they have authenticated.</dd>
          <dt><code>state=</code></dt>
          <dd>A random string that will be passed back after authentication to prevent CSRF attacks.</dd>
        </dl>

        <h2>The user is redirected back to the URI specified</h2>
        <p>Once authentication is complete the user is sent back to your site with a couple of query parameters:</p>
        <dl>
          <dt><code>state=</code></dt>
          <dd>The random string you originally sent, check this matches before continuing.</dd>
          <dt><code>code=</code></dt>
          <dd>A string you will need to verify to complete authentication.</dd>
        </dl>

        <h2>Verify the code</h2>
        <p>Make a <code>POST</code> request to <code>{{ .ThisURI }}/auth</code> to verify the
          code you recieved. In return you will get the web address for the
          authenticated user.</p>

        <pre><code>POST {{ .ThisURI }}/auth HTTP/1.1
Content-Type: application/x-www-form-urlencoded;charset=UTF-8
Accept: application/json

code=kgnn18riem3pssk74&
redirect_uri=https://example.com/callback&
client_id=https://example.com/</code></pre>

        <p>Will, if correct, receive a JSON response with a value <code>"me"</code>.</p>

        <pre><code>HTTP/1.1 200 OK
Content-Type: application/json

{
  "me": "https://john.doe/"
}</code></pre>

        <p>Store the web address in a secure session and log the user in. You are done.</p>
      </div>
    </section>

    <footer>
      <div class="container">
        <p>The source code for this project is on <a href="https://github.com/hawx/relme-auth">GitHub</a>.</p>
        <p>For more information on RelMeAuth, or other implementations, read <a href="https://indieweb.org/RelMeAuth">its IndieWeb wiki entry.</a></p>
      </div>
    </footer>
  </body>
</html>`
