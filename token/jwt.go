package token

import (
	"strings"
	"encoding/json"
	"encoding/base64"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/rand"
)

type JsonWebToken struct {
	Header Header
	Data Data
}

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type Data struct {
	Sub string `json:"sub"`
}

func NewJWT(sub string) *JsonWebToken {
	return &JsonWebToken{
		Header: Header{
			Alg: "RS256",
			Typ: "JWT",
		},
		Data: Data{
			Sub: sub,
		},
	}
}

func (jwt *JsonWebToken) Encode(privateKey *rsa.PrivateKey) (string, error) {
	jsonHeader, err := json.Marshal(jwt.Header)
	if err != nil {
		return "", err
	}
	encodedHeader := base64.URLEncoding.EncodeToString(jsonHeader)

	jsonData, err := json.Marshal(jwt.Data)
	if err != nil {
		return "", err
	}
	encodedData := base64UrlEncode(jsonData)

	signature, err := RSASHA256(encodedHeader + "." + encodedData, privateKey)
	if err != nil {
		return "", err
	}
	
	return encodedHeader + "." + encodedData + "." + signature, nil
}

func RSASHA256(data string, key *rsa.PrivateKey) (string, error) {
	hashed := sha256.Sum256([]byte(data))
	
	signed, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	return base64UrlEncode(signed), err
}

func base64UrlEncode(data []byte) string {
	r := strings.NewReplacer(
		"+", "-",
		"/", "_")

	swapped := r.Replace(base64.StdEncoding.EncodeToString(data))
	return strings.TrimRight(swapped, "=")
}
