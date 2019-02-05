package handler

import (
	"net/http"
	"net/url"

	"github.com/peterhellberg/link"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/microformats"
	"hawx.me/code/relme-auth/strategy"
)

// Auth takes the chosen provider and initiates authentication by redirecting
// the user to the 3rd party. It takes a number of parameters:
//
//   - me: URL originally entered of who we are trying to authenticate
//   - provider: 3rd party authentication provider that was chosen
//   - profile: URL expected to be matched by the provider
//   - redirect_uri: final URI to redirect to when auth is finished
func Auth(authStore data.SessionStore, strategies strategy.Strategies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			me          = r.FormValue("me")
			provider    = r.FormValue("provider")
			profile     = r.FormValue("profile")
			redirectURI = r.FormValue("redirect_uri")

			chosenStrategy strategy.Strategy
			ok             bool
		)

		session, ok := authStore.Get(me)
		if !ok {
			http.Error(w, "you need to start at the start", http.StatusBadRequest)
			return
		}

		if session.RedirectURI != redirectURI || !verifyRedirectURI(session.ClientID, session.RedirectURI) {
			http.Error(w, "redirect_uri is untrustworthy", http.StatusBadRequest)
			return
		}

		for _, s := range strategies {
			if s.Name() == provider {
				chosenStrategy = s
				ok = true
				break
			}
		}

		if !ok {
			http.Error(w, "No rel=\"me\" links on your profile match a known provider", http.StatusBadRequest)
			return
		}

		redirectURL, err := chosenStrategy.Redirect(me)
		if err != nil {
			http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
			return
		}

		session.Provider = provider
		session.ProfileURI = profile

		authStore.Update(session)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})
}

func verifyRedirectURI(clientID, redirect string) bool {
	clientURI, err := url.Parse(clientID)
	if err != nil {
		return false
	}

	redirectURI, err := url.Parse(redirect)
	if err != nil {
		return false
	}

	if clientURI.Scheme == redirectURI.Scheme && clientURI.Host == redirectURI.Host {
		return true
	}

	clientResp, err := http.Get(clientID)
	if err != nil {
		return false
	}
	defer clientResp.Body.Close()

	if clientResp.StatusCode < 200 && clientResp.StatusCode >= 300 {
		return false
	}

	var whitelist []string

	if whitelistedRedirect, ok := link.ParseResponse(clientResp)["redirect_uri"]; ok {
		whitelist = append(whitelist, whitelistedRedirect.URI)
	}

	whitelist = append(whitelist, microformats.RedirectURIs(clientResp.Body)...)

	for _, candidate := range whitelist {
		if candidate == redirect {
			return true
		}
	}

	return false
}
