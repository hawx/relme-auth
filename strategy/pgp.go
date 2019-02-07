package strategy

import (
	"bytes"
	"crypto/rand"
	"errors"
	"net/http"
	"net/url"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

type authPGP struct {
	AuthURL    string
	ClientID   string
	Store      strategyStore
	httpClient *http.Client
}

// PGP provides a strategy for authenticating with a pgpkey.
func PGP(store strategyStore, baseURI, id string, httpClient *http.Client) Strategy {
	return &authPGP{
		AuthURL:    baseURI + "/pgp/authorize",
		ClientID:   id,
		Store:      store,
		httpClient: httpClient,
	}
}

func (authPGP) Name() string {
	return "pgp"
}

func (authPGP) Match(me *url.URL) bool {
	return me.String() == "pgp"
}

func (strategy *authPGP) Redirect(expectedLink string) (redirectURL string, err error) {
	state, err := strategy.Store.Insert(expectedLink)
	if err != nil {
		return "", err
	}
	challenge, err := randomString(40)
	if err != nil {
		return "", err
	}
	if err := strategy.Store.Set(expectedLink, challenge); err != nil {
		return "", err
	}

	query := url.Values{
		"client_id": {strategy.ClientID},
		"state":     {state},
		"challenge": {challenge},
	}

	return strategy.AuthURL + "?" + query.Encode(), nil
}

func (strategy *authPGP) Callback(form url.Values) (string, error) {
	expectedURL, ok := strategy.Store.Claim(form.Get("state"))
	if !ok {
		return "", errors.New("how did you get here?")
	}
	challenge, ok := strategy.Store.Claim(expectedURL)
	if !ok {
		return "", errors.New("how did you get here?")
	}

	if err := verify(strategy.httpClient, expectedURL, form.Get("signed"), challenge); err != nil {
		return "", ErrUnauthorized
	}

	return expectedURL, nil
}

func verify(httpClient *http.Client, keyURL, signed, challenge string) error {
	resp, err := httpClient.Get(keyURL)
	if err != nil {
		return errors.New("could not get file: " + err.Error())
	}
	defer resp.Body.Close()

	keyRing, err := openpgp.ReadArmoredKeyRing(resp.Body)
	if err != nil {
		return errors.New("could not read key: " + err.Error())
	}

	blk, rest := clearsign.Decode([]byte(signed))
	if len(rest) != 0 {
		return errors.New("more data than expected")
	}

	if blk == nil || !bytes.Equal(blk.Bytes, []byte(challenge)) {
		return errors.New("challenge not correct")
	}

	_, err = openpgp.CheckDetachedSignature(keyRing, bytes.NewBuffer(blk.Bytes), blk.ArmoredSignature.Body)
	return err
}

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"

// RandomString produces a random string of n characters.
func randomString(n int) (string, error) {
	bytes, err := randomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

func randomBytes(length int) (b []byte, err error) {
	b = make([]byte, length)
	_, err = rand.Read(b)
	return
}
