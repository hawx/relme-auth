package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/internal/config"
	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/handler"
	"hawx.me/code/relme-auth/internal/microformats"
	"hawx.me/code/relme-auth/internal/random"
	"hawx.me/code/relme-auth/internal/strategy"
	"hawx.me/code/route"
	"hawx.me/code/serve"
)

func printHelp() {
	fmt.Println(`Usage: relme-auth [options]

  relme-auth is a web service for authenticating with 3rd party
  auth providers.

  The providers implemented are:
   * GitHub
   * Flickr
   * Twitter
   * PGP

 CONFIGURATION
   --config PATH='./config.toml'
     Configuration file to use, this defines the secrets for
     communicating with 3rd party authentication providers.

   --base-url URL='http://localhost:8080'
     Where this app is going to be accessible from.

   --cookie-secret SECRET
     A base64 encoded string to use for authenticating sessions.
     It is recommended to use 32 or 64 bytes for this value.

   --true
     Use the fake 'true' authentication provider. This should
     only be used locally for testing as it says everyone is
     authenticated!

 DATA
   --db PATH
      Use the sqlite database at the given path.

 SERVE
   Will use a systemd.socket if configured to do so.

   --port PORT='8080'
      Serve on given port.

   --socket SOCK
      Serve at given socket, instead.`)
}

func main() {
	var (
		port         = flag.String("port", "8080", "Port to run on")
		socket       = flag.String("socket", "", "Socket to run on")
		baseURL      = flag.String("base-url", "http://localhost:8080", "Where this is running")
		configPath   = flag.String("config", "./config.toml", "Path to config file")
		dbPath       = flag.String("db", "", "Path to database")
		cookieSecret = flag.String("cookie-secret", "", "Secret to authenticate sessions with")
		useTrue      = flag.Bool("true", false, "Use the fake 'true' auth provider")
		webPath      = flag.String("web-path", "web", "Path to web/ directory")
	)
	flag.Usage = func() { printHelp() }
	flag.Parse()

	conf, err := config.Read(*configPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	// use default values from DefaultTransport
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout:   10 * time.Second,
		Transport: tr,
	}

	codeGenerator := random.Generator(20)
	tokenGenerator := random.Generator(40)

	secret, err := base64.StdEncoding.DecodeString(*cookieSecret)
	if err != nil || len(secret) == 0 {
		fmt.Println("could not base64 decode cookie-secret:", err)
		return
	}

	cookies := sessions.NewCookieStore(secret)
	cookies.Options.HttpOnly = true
	cookies.Options.SameSite = http.SameSiteStrictMode
	cookies.Options.Secure = strings.HasPrefix(*baseURL, "https://")

	database, err := data.Open(*dbPath, httpClient, cookies, data.Expiry{
		Session: 5 * time.Minute,
		Code:    60 * time.Second,
		Client:  24 * time.Hour,
		Profile: 7 * 24 * time.Hour,
		Login:   8 * time.Hour,
	})
	if err != nil {
		fmt.Println("could not open database:", err)
		return
	}
	defer database.Close()

	templates, err := template.ParseGlob(*webPath + "/template/*")
	if err != nil {
		fmt.Println("could not load templates:", err)
		return
	}

	route.Handle("/callback/continue", handler.Continue(database, codeGenerator))

	var strategies strategy.Strategies
	if *useTrue {
		trueStrategy := strategy.True(*baseURL)
		strategies = append(strategies, trueStrategy)

		route.Handle("/callback/true", handler.Callback(database, trueStrategy, codeGenerator))

	} else {
		pgpDatabase, _ := data.Strategy("pgp")
		pgpStrategy := strategy.PGP(pgpDatabase, *baseURL, "", httpClient)
		route.Handle("/callback/pgp", handler.Callback(database, pgpStrategy, codeGenerator))
		strategies = append(strategies, pgpStrategy)

		if conf.Flickr != nil {
			flickrDatabase, _ := data.Strategy("flickr")
			flickrStrategy := strategy.Flickr(*baseURL, flickrDatabase, conf.Flickr.ID, conf.Flickr.Secret, httpClient)
			route.Handle("/callback/flickr", handler.Callback(database, flickrStrategy, codeGenerator))
			strategies = append(strategies, flickrStrategy)
		}

		if conf.GitHub != nil {
			gitHubDatabase, _ := data.Strategy("github")
			gitHubStrategy := strategy.GitHub(gitHubDatabase, conf.GitHub.ID, conf.GitHub.Secret)
			route.Handle("/callback/github", handler.Callback(database, gitHubStrategy, codeGenerator))
			strategies = append(strategies, gitHubStrategy)
		}

		if conf.Twitter != nil {
			twitterDatabase, _ := data.Strategy("twitter")
			twitterStrategy := strategy.Twitter(*baseURL, twitterDatabase, conf.Twitter.ID, conf.Twitter.Secret, httpClient)
			route.Handle("/callback/twitter", handler.Callback(database, twitterStrategy, codeGenerator))
			strategies = append(strategies, twitterStrategy)
		}
	}

	route.Handle("/auth", mux.Method{
		"GET":  handler.Choose(*baseURL, database, strategies, templates),
		"POST": handler.Verify(database),
	})
	route.Handle("/auth/start", mux.Method{
		"GET": handler.Auth(database, strategies, httpClient),
	})

	route.Handle("/token", handler.Token(database, tokenGenerator))
	route.Handle("/pgp/authorize", handler.PGP(templates))

	route.Handle("/", handler.Example(*baseURL, conf, cookies, database, templates))
	route.Handle("/callback", handler.ExampleCallback(*baseURL, cookies))
	route.Handle("/sign-out", handler.ExampleSignOut(*baseURL, cookies))
	route.Handle("/revoke", handler.ExampleRevoke(*baseURL, cookies, database))
	route.Handle("/privacy", handler.ExamplePrivacy(*baseURL, cookies, templates))
	route.Handle("/forget", handler.ExampleForget(*baseURL, cookies, database))
	route.Handle("/generate", handler.ExampleGenerate(*baseURL, cookies, tokenGenerator, database, templates))

	relMe := &microformats.RelMe{Client: httpClient, NoRedirectClient: noRedirectClient}

	route.Handle("/ws", handler.WebSocket(strategies, database, relMe))
	route.Handle("/public/*path", http.StripPrefix("/public", http.FileServer(http.Dir(*webPath+"/static"))))

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      context.ClearHandler(route.Default),
	}

	serve.Server(*port, *socket, srv)
}
