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
		port           = flag.String("port", "8080", "Port to run on")
		socket         = flag.String("socket", "", "Socket to run on")
		configPath     = flag.String("config", "./config.toml", "Path to config file")
		privateKeyPath = flag.String("private-key", "./priv.pem", "Path to private key in pem format")
		useTrue        = flag.Bool("true", false, "Use the fake 'true' auth provider")
	)
	flag.Parse()

	conf, err := config.Read(*configPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	privateKey, err := config.ReadPrivateKey(*privateKeyPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	authStore := memory.NewStore()

	if *useTrue {
		trueStrategy := strategy.True()

		route.Handle("/auth", mux.Method{
			"GET":  handler.Choose(authStore, strategy.Strategies{trueStrategy}),
			"POST": handler.Verify(authStore),
		})
		route.Handle("/auth/start", mux.Method{
			"GET": handler.Auth(authStore, strategy.Strategies{trueStrategy}),
		})
		route.Handle("/oauth/callback/true", handler.Callback(privateKey, authStore, trueStrategy))
		route.Handle("/*rest", handler.Example())

	} else {
		flickrStrategy := strategy.Flickr(authStore, conf.Flickr.Id, conf.Flickr.Secret)
		gitHubStrategy := strategy.GitHub(authStore, conf.GitHub.Id, conf.GitHub.Secret)
		twitterStrategy := strategy.Twitter(authStore, conf.Twitter.Id, conf.Twitter.Secret)

		strategies := strategy.Strategies{
			flickrStrategy,
			gitHubStrategy,
			twitterStrategy,
		}

		route.Handle("/auth", mux.Method{
			"GET":  handler.Choose(authStore, strategies),
			"POST": handler.Verify(authStore),
		})
		route.Handle("/auth/start", mux.Method{
			"GET": handler.Auth(authStore, strategies),
		})
		route.Handle("/oauth/callback/flickr", handler.Callback(privateKey, authStore, flickrStrategy))
		route.Handle("/oauth/callback/github", handler.Callback(privateKey, authStore, gitHubStrategy))
		route.Handle("/oauth/callback/twitter", handler.Callback(privateKey, authStore, twitterStrategy))
		route.Handle("/*rest", handler.Example())
	}

	serve.Serve(*port, *socket, route.Default)
}
