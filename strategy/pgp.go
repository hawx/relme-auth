package strategy

import (
	"bytes"
	"crypto/rand"
	"errors"
	"os"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
	"hawx.me/code/relme-auth/data"
)

type authPGP struct {
	Store data.StrategyStore
}

func PGP(store data.StrategyStore) *authPGP {
	return &authPGP{
		Store: store,
	}
}

func (authPGP) Name() string {
	return "pgp"
}

func (strategy *authPGP) Challenge(me string) string {
	s, _ := randomString(20)

	strategy.Store.Set(me, s)
	return s
}

func (strategy *authPGP) Verify(me string, signed string) (string, error) {
	challenge, ok := strategy.Store.Claim(me)
	if !ok {
		return "", errors.New("How did you get here?")
	}

	public, err := os.Open("public.asc")
	if err != nil {
		return "", errors.New("could not get file: " + err.Error())
	}
	keyRing, err := openpgp.ReadArmoredKeyRing(public)
	if err != nil {
		return "", errors.New("could not read key: " + err.Error())
	}

	blk, rest := clearsign.Decode([]byte(signed))
	if len(rest) != 0 {
		return "", ErrUnauthorized
	}

	if !bytes.Equal(blk.Bytes, []byte(challenge)) {
		return "", ErrUnauthorized
	}

	_, err = openpgp.CheckDetachedSignature(keyRing, bytes.NewBuffer(blk.Bytes), blk.ArmoredSignature.Body)
	if err != nil {
		return "", ErrUnauthorized
	}

	return me, nil
}

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"

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
