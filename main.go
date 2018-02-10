package main

import (
	"net/url"
	"net/http"
	"fmt"
	"flag"
	"hawx.me/code/serve"
	"hawx.me/code/route"
	"hawx.me/code/relme"
	"hawx.me/code/relme-auth/strategy"
	"hawx.me/code/relme-auth/state"
	"github.com/BurntSushi/toml"
)

type Config struct {
	GitHub *strategyConfig `toml:"github"`
}

type strategyConfig struct {
	Id string `toml:"id"`
	Secret string `toml:"secret"`
}

func readConfig(path string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(path, &conf)
	return conf, err
}

func main() {
	var (
		port = flag.String("port", "8080", "Port to run on")
		socket = flag.String("socket", "", "Socket to run on")
		config = flag.String("config", "./config.toml", "Path to config file")
	)
	flag.Parse()

	conf, err := readConfig(*config)
	if err != nil {
		fmt.Println(err)
		return
	}
	
	gitHubStrategy := strategy.GitHub(conf.GitHub.Id, conf.GitHub.Secret)
	
	strategies := []strategy.Strategy{
		gitHubStrategy,
	}

	authStore := state.Store()
	
	route.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<body>
  <form action="/authenticate" method="POST">
    <label for="me">Web Address:</label>
    <input type="url" id="me" name="me" />
    <button type="submit">Sign-in</button>
  </form>
</body>
</html>
`)
	})

	route.HandleFunc("/authenticate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(415)
			return
		}

		me := r.FormValue("me")

		verifiedLinks, _ := relme.FindVerified(me)
		if chosenStrategy, expectedLink, ok := findStrategy(verifiedLinks, strategies); ok {
			state, err := authStore.Insert(expectedLink)
			if err != nil {
				http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
				return
			}
			
			http.Redirect(w, r, chosenStrategy.Redirect(state), http.StatusFound)
			return
		}

		http.Redirect(w, r, "/no-strategies", http.StatusFound)
	})

	route.HandleFunc("/oauth/callback/github", func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		state := r.FormValue("state")

		expectedURL, ok := authStore.Claim(state)
		if !ok {
			http.Error(w, "How did you get here?", http.StatusInternalServerError)
			return
		}

		userProfileURL, err := gitHubStrategy.Callback(code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if userProfileURL != expectedURL {
			http.Error(w, "You are not the user I was expecting", http.StatusUnauthorized)
			return
		}

		fmt.Fprint(w, "Here is a JWT")
	})
	
	serve.Serve(*port, *socket, route.Default)
}

func findStrategy(verifiedLinks []string, strategies []strategy.Strategy) (s strategy.Strategy, expectedLink string, ok bool) {
	for _, link := range verifiedLinks {
		fmt.Printf("me=%s\n", link)
		linkURL, _ := url.Parse(link)
		
		for _, strategy := range strategies {
			if strategy.Match(linkURL) {
				fmt.Printf("Can authenticate with %s\n", link)
				return strategy, link, true
			}
		}
	}

	return
}

