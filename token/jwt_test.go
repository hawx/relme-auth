package token

import (
	"testing"
	"hawx.me/code/assert"
	"io/ioutil"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/dvsekhvalnov/jose2go"
)

func TestJWT(t *testing.T) {
	expected := `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJqb2huQGV4YW1wbGUuY29tIn0.O6kUG9ez2UHb-84-UzJVNThVmVc4-c9ZN99rksW6In7n82m3kfrMz7rx30Nyv2hO_HiPmK8bFRJm7FA9nQyHKTrguIb1H-Y5EjnVWRW_EIjmT1h-M1K_14SFTpvH0NRN3JjpwcTtv60IWK7bY9rJmFZtQER8EgQfTHZk24xYxEs`
	
	privateKeyBytes, _ := ioutil.ReadFile("priv.pem")
	privateKey := ReadPrivate(privateKeyBytes)

	publicKeyBytes, _ := ioutil.ReadFile("pub.pem")
	publicKey := ReadPublic(publicKeyBytes)
	
	jwt := NewJWT("john@example.com")
	
	encoded, err := jwt.Encode(privateKey)
	
	assert.Nil(t, err)
	assert.Equal(t, encoded, expected)

	// check decoding works
	payload, headers, err := jose.Decode(encoded, publicKey)
	assert.Nil(t, err)
	assert.Equal(t, headers["alg"], "RS256")
	assert.Equal(t, headers["typ"], "JWT")
	assert.Equal(t, payload, `{"sub":"john@example.com"}`)
}

func ReadPrivate(raw []byte) *rsa.PrivateKey {
	encoded, _ := pem.Decode(raw)
	parsedKey, _ := x509.ParsePKCS1PrivateKey(encoded.Bytes)
	return parsedKey
}

func ReadPublic(raw []byte) *rsa.PublicKey  {
	encoded, _ := pem.Decode(raw)
	cert, _ := x509.ParseCertificate(encoded.Bytes)
	key, _ := cert.PublicKey.(*rsa.PublicKey)
	return key
}
