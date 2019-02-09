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

type pgpData struct {
	me        string
	profile   string
	challenge string
}

type authPGP struct {
	authURL    string
	clientID   string
	store      strategyStore
	httpClient *http.Client
}

// PGP provides a strategy for authenticating with a pgpkey.
func PGP(store strategyStore, baseURI, id string, httpClient *http.Client) Strategy {
	return &authPGP{
		authURL:    baseURI + "/pgp/authorize",
		clientID:   id,
		store:      store,
		httpClient: httpClient,
	}
}

func (authPGP) Name() string {
	return "pgp"
}

func (authPGP) Match(profile *url.URL) bool {
	return profile.String() == "pgp"
}

func (strategy *authPGP) Redirect(me, profile string) (redirectURL string, err error) {
	challenge, err := randomString(40)
	if err != nil {
		return "", err
	}

	state, err := strategy.store.Insert(pgpData{
		me:        me,
		profile:   profile,
		challenge: challenge,
	})
	if err != nil {
		return "", err
	}

	query := url.Values{
		"client_id": {strategy.clientID},
		"state":     {state},
		"challenge": {challenge},
	}

	return strategy.authURL + "?" + query.Encode(), nil
}

func (strategy *authPGP) Callback(form url.Values) (string, error) {
	data, ok := strategy.store.Claim(form.Get("state"))
	if !ok {
		return "", errors.New("how did you get here? 1")
	}
	fdata := data.(pgpData)

	if err := verify(strategy.httpClient, fdata.profile, form.Get("signed"), fdata.challenge); err != nil {
		return "", ErrUnauthorized
	}

	return fdata.me, nil
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
