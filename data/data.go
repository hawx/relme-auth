// Package data defines the structures required for storing session state
//
// See subpackages for actual implementations
package data

import (
	"crypto/rand"
)

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"

// RandomString produces a random string of n characters.
func RandomString(n int) (string, error) {
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
