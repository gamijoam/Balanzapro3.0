// Package argon2id implements the server-side password verifier (ADR-026).
// It is not a login service and must not be used for low-entropy PINs
// without additional protections and rate limiting.
package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memoryKiB        = 65536
	iterations       = 3
	lanes            = 1
	saltSize         = 16
	digestSize       = 32
	maxPasswordBytes = 1024
	envelope         = "$argon2id$v=19$m=65536,t=3,p=1$"
)

var ErrInvalidPassword = errors.New("identity: invalid password length")
var encoding = base64.RawStdEncoding.Strict()

// Hash derives a verifier with a new cryptographically random salt.
func Hash(password []byte) (string, error) {
	if len(password) == 0 || len(password) > maxPasswordBytes {
		return "", ErrInvalidPassword
	}
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := argon2.IDKey(password, salt, iterations, memoryKiB, lanes, digestSize)
	defer clear(digest)
	return envelope + encoding.EncodeToString(salt) + "$" + encoding.EncodeToString(digest), nil
}

// ValidEnvelope checks the exact supported PHC profile before allocating
// Argon2 memory. Future profiles require an explicitly bounded parser.
func ValidEnvelope(verifier string) bool {
	_, _, ok := parse(verifier)
	return ok
}

func parse(verifier string) ([]byte, []byte, bool) {
	if len(verifier) > 512 || !strings.HasPrefix(verifier, envelope) {
		return nil, nil, false
	}
	a, b, ok := strings.Cut(strings.TrimPrefix(verifier, envelope), "$")
	if !ok || len(a) != encoding.EncodedLen(saltSize) || len(b) != encoding.EncodedLen(digestSize) {
		return nil, nil, false
	}
	salt, err := encoding.DecodeString(a)
	if err != nil || len(salt) != saltSize || encoding.EncodeToString(salt) != a {
		return nil, nil, false
	}
	digest, err := encoding.DecodeString(b)
	if err != nil || len(digest) != digestSize || encoding.EncodeToString(digest) != b {
		return nil, nil, false
	}
	return salt, digest, true
}

// Verify compares derived bytes in constant time for a valid envelope.
// Authentication must separately implement throttling and uniform failures.
func Verify(password []byte, verifier string) bool {
	if len(password) == 0 || len(password) > maxPasswordBytes {
		return false
	}
	salt, want, ok := parse(verifier)
	if !ok {
		return false
	}
	actual := argon2.IDKey(password, salt, iterations, memoryKiB, lanes, digestSize)
	defer clear(actual)
	return subtle.ConstantTimeCompare(actual, want) == 1
}
