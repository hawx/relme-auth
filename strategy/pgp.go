package strategy

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
	"hawx.me/code/relme-auth/data"
)

type authPGP struct {
	AuthURL  string
	ClientID string
	Store    strategyStore
}

// PGP provides a strategy for authenticating with a pgpkey.
func PGP(store strategyStore, baseURI, id string) Strategy {
	return &authPGP{
		AuthURL:  baseURI + "/pgp/authorize",
		ClientID: id,
		Store:    store,
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
	challenge, err := data.RandomString(40)
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

	if err := verify(expectedURL, form.Get("signed"), challenge); err != nil {
		return "", ErrUnauthorized
	}

	return expectedURL, nil
}

func verify(keyURL, signed, challenge string) error {
	resp, err := http.Get(keyURL)
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
