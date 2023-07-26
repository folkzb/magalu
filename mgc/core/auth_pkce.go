package core

import (
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"strings"
	"time"
)

type codeVerifier struct {
	value string
}

const (
	length = 32
)

func base64URLEncode(str []byte) string {
	encoded := base64.StdEncoding.EncodeToString(str)
	encoded = strings.Replace(encoded, "+", "-", -1)
	encoded = strings.Replace(encoded, "/", "_", -1)
	encoded = strings.Replace(encoded, "=", "", -1)
	return encoded
}

func newVerifier() (*codeVerifier, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = byte(r.Intn(255))
	}
	return newCodeVerifierFromBytes(b)
}

func newCodeVerifierFromBytes(b []byte) (*codeVerifier, error) {
	return &codeVerifier{
		value: base64URLEncode(b),
	}, nil
}

func (v *codeVerifier) CodeChallengeS256() string {
	h := sha256.New()
	h.Write([]byte(v.value))
	return base64URLEncode(h.Sum(nil))
}
