package main

import (
	"flag"
	"fmt"

	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/config"
	"hawx.me/code/relme-auth/handler"
	"hawx.me/code/relme-auth/store/memory"
	"hawx.me/code/relme-auth/strategy"
	"hawx.me/code/route"
	"hawx.me/code/serve"
)

func main() {
	var (
		port       = flag.String("port", "8080", "Port to run on")
		socket     = flag.String("socket", "", "Socket to run on")
		configPath = flag.String("config", "./config.toml", "Path to config file")
		useTrue    = flag.Bool("true", false, "Use the fake 'true' auth provider")
	)
	flag.Parse()

	conf, err := config.Read(*configPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	authStore := memory.NewStore()
	var strategies strategy.Strategies

	if *useTrue {
		trueStrategy := strategy.True()
		strategies = strategy.Strategies{trueStrategy}

		route.Handle("/oauth/callback/true", handler.Callback(authStore, trueStrategy))

	} else {
		flickrStrategy := strategy.Flickr(authStore, conf.Flickr.Id, conf.Flickr.Secret)
		gitHubStrategy := strategy.GitHub(authStore, conf.GitHub.Id, conf.GitHub.Secret)
		twitterStrategy := strategy.Twitter(authStore, conf.Twitter.Id, conf.Twitter.Secret)

		strategies = strategy.Strategies{
			flickrStrategy,
			gitHubStrategy,
			twitterStrategy,
		}

		route.Handle("/oauth/callback/flickr", handler.Callback(authStore, flickrStrategy))
		route.Handle("/oauth/callback/github", handler.Callback(authStore, gitHubStrategy))
		route.Handle("/oauth/callback/twitter", handler.Callback(authStore, twitterStrategy))
	}

	route.Handle("/auth", mux.Method{
		"GET":  handler.Choose(authStore, strategies),
		"POST": handler.Verify(authStore),
	})
	route.Handle("/auth/start", mux.Method{
		"GET": handler.Auth(authStore, strategies),
	})
	route.Handle("/*rest", handler.Example())

	serve.Serve(*port, *socket, route.Default)
}
