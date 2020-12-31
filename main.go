package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"hawx.me/code/relme-auth/internal/config"
	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/random"
	"hawx.me/code/relme-auth/internal/server"
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

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
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

	serve.Server(*port, *socket, &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler: server.New(
			database,
			codeGenerator,
			*baseURL,
			httpClient,
			conf,
			*useTrue,
			*webPath,
			templates,
			cookies,
			tokenGenerator,
			noRedirectClient,
		),
	})
}
