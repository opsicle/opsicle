package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	hashMemory  = 64 * 1024 // 64 MB
	hashTime    = 3
	hashThreads = 4
	hashKeyLen  = 32
	hashSaltLen = 16
)

func IsPasswordValid(password string) (bool, error) {
	var errs []error

	if len(password) < 12 {
		errs = append(errs, errors.New("password must be at least 12 characters long"))
	}

	var hasUpper, hasLower, hasNumber, hasSymbol bool

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasNumber = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSymbol = true
		}
	}

	if !hasUpper {
		errs = append(errs, errors.New("password must contain at least one uppercase letter"))
	}
	if !hasLower {
		errs = append(errs, errors.New("password must contain at least one lowercase letter"))
	}
	if !hasNumber {
		errs = append(errs, errors.New("password must contain at least one number"))
	}
	if !hasSymbol {
		errs = append(errs, errors.New("password must contain at least one symbol or punctuation character"))
	}
	if len(errs) > 0 {
		return false, errors.Join(errs...)
	}
	return true, nil
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, hashSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, hashTime, hashMemory, hashThreads, hashKeyLen)
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		hashMemory, hashTime, hashThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

func ValidatePassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false
	}

	var mem uint32
	var t uint32
	var p uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &t, &p)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, t, mem, p, uint32(len(expectedHash)))

	return subtleCompare(hash, expectedHash)
}

func subtleCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
