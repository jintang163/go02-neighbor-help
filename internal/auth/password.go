package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

const (
	defaultIterations = 10000
	defaultSaltLen    = 16
)

type PasswordHasher struct {
	iterations int
	saltLen    int
	now        func() time.Time
}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		iterations: defaultIterations,
		saltLen:    defaultSaltLen,
		now:        time.Now,
	}
}

func (h *PasswordHasher) Hash(password string) (salt, hash string, iterations int, err error) {
	if password == "" {
		return "", "", 0, errors.New("password must not be empty")
	}
	saltBytes := make([]byte, h.saltLen)
	if _, err = rand.Read(saltBytes); err != nil {
		return "", "", 0, fmt.Errorf("generate salt: %w", err)
	}
	dk := deriveKey([]byte(password), saltBytes, h.iterations)
	return base64.StdEncoding.EncodeToString(saltBytes),
		base64.StdEncoding.EncodeToString(dk),
		h.iterations,
		nil
}

func (h *PasswordHasher) Verify(password, saltB64, hashB64 string, iterations int) bool {
	saltBytes, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return false
	}
	expected, err := base64.StdEncoding.DecodeString(hashB64)
	if err != nil {
		return false
	}
	if iterations <= 0 {
		iterations = defaultIterations
	}
	dk := deriveKey([]byte(password), saltBytes, iterations)
	return hmac.Equal(dk, expected)
}

func deriveKey(password, salt []byte, iterations int) []byte {
	h := sha256.New()
	h.Write(salt)
	h.Write(password)
	dk := h.Sum(nil)
	for i := 1; i < iterations; i++ {
		h.Reset()
		h.Write(dk)
		h.Write(password)
		dk = h.Sum(nil)
	}
	return dk
}

func (h *PasswordHasher) MustHash(password string) (salt, hash string, iterations int) {
	s, hs, it, err := h.Hash(password)
	if err != nil {
		panic(err)
	}
	return s, hs, it
}
