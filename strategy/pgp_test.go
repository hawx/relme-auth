package strategy

import (
	"bytes"
	"os"
	"testing"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data/memory"
)

func TestPGP(t *testing.T) {
	assert := assert.New(t)

	private, _ := os.Open("private.asc")
	defer private.Close()
	keyRing, _ := openpgp.ReadArmoredKeyRing(private)

	strategy := PGP(memory.New())

	challenge := strategy.Challenge("https://example.com")

	signedMsg := bytes.NewBuffer(nil)
	dec, err := clearsign.Encode(signedMsg, keyRing[0].PrivateKey, nil)
	if assert.Nil(err) {
		dec.Write([]byte(challenge))
		dec.Close()

		me, err := strategy.Verify("https://example.com", signedMsg.String())
		assert.Nil(err)
		assert.Equal("https://example.com", me)
	}
}

func TestPGPWithWrongKey(t *testing.T) {
	assert := assert.New(t)

	private, _ := os.Open("other_private.asc")
	defer private.Close()
	keyRing, _ := openpgp.ReadArmoredKeyRing(private)

	strategy := PGP(memory.New())

	challenge := strategy.Challenge("https://example.com")

	signedMsg := bytes.NewBuffer(nil)
	dec, err := clearsign.Encode(signedMsg, keyRing[0].PrivateKey, nil)
	if assert.Nil(err) {
		dec.Write([]byte(challenge))
		dec.Close()

		me, err := strategy.Verify("https://example.com", signedMsg.String())
		assert.Equal(ErrUnauthorized, err)
		assert.Equal("", me)
	}
}
