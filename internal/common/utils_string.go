package common

import "crypto/rand"

const randomStringCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	charsetLength := byte(len(randomStringCharset))

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i := range bytes {
		bytes[i] = randomStringCharset[bytes[i]%charsetLength]
	}

	return string(bytes), nil
}
