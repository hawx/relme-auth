package state

import (
	"crypto/rand"
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" // 52 possibilities
	letterIdxBits = 6                    // 6 bits to represent 64 possibilities / indexes
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
)

func secureRandomAlphaString(length int) (randomString string, err error) {
	result := make([]byte, length)
	bufferSize := int(float64(length)*1.3)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			randomBytes, err = secureRandomBytes(bufferSize)
			if err != nil {
				return
			}
		}
		if idx := int(randomBytes[j%length] & letterIdxMask); idx < len(letterBytes) {
			result[i] = letterBytes[idx]
			i++
		}
	}

	return string(result), nil
}

// secureRandomBytes returns the requested number of bytes using crypto/rand
func secureRandomBytes(length int) (randomBytes []byte, err error) {
	randomBytes = make([]byte, length)
	_, err = rand.Read(randomBytes)
	return 
}
