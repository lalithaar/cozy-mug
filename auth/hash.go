package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	// defaultPBKDF2Iterations is tuned for ordinary login cost in a small app.
	// It is intentionally exposed only through the prefix so verification of
	// old hashes remains possible if the value changes later.
	defaultPBKDF2Iterations = 600000

	pbkdf2SaltBytes  = 16
	pbkdf2KeyBytes   = 32
	maxPasswordBytes = 4096
)

// PBKDF2Hasher is the default password hasher. It uses the standard
// library's PBKDF2 implementation and stores enough information in the hash to
// let later code verify (and rehash if required) old hashes.
type PBKDF2Hasher struct{}

// Hash returns "$pbkdf2-sha256$<iterations>$<salt>$<key>".
func (PBKDF2Hasher) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("auth: password must not be empty")
	}
	if len(password) > maxPasswordBytes {
		return "", errors.New("auth: password is too long")
	}
	salt := make([]byte, pbkdf2SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: random salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, defaultPBKDF2Iterations, pbkdf2KeyBytes)
	if err != nil {
		return "", fmt.Errorf("auth: key derivation: %w", err)
	}
	return fmt.Sprintf("$pbkdf2-sha256$%d$%s$%s",
		defaultPBKDF2Iterations,
		base64.RawURLEncoding.EncodeToString(salt),
		base64.RawURLEncoding.EncodeToString(key)), nil
}

// Compare reports whether password matches hash.
func (PBKDF2Hasher) Compare(hash, password string) bool {
	if hash == "" || password == "" {
		return false
	}
	parts := strings.Split(hash, "$")
	// Expected: "", "pbkdf2-sha256", iterations, salt, key
	if len(parts) != 5 || parts[0] != "" || parts[1] != "pbkdf2-sha256" {
		return false
	}
	iterations := 0
	if _, err := fmt.Sscanf(parts[2], "%d", &iterations); err != nil || iterations <= 0 {
		return false
	}
	salt, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	want, err := base64.RawURLEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	if len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return false
	}
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// DefaultHasher is used unless the app supplies another Hasher.
var DefaultHasher Hasher = PBKDF2Hasher{}

// HashPassword hashes using the default hasher.
func HashPassword(password string) (string, error) { return DefaultHasher.Hash(password) }

// CheckPassword verifies using the default hasher.
func CheckPassword(hash, password string) bool { return DefaultHasher.Compare(hash, password) }

// Hasher is the small seam that lets an app use bcrypt/argon2 if required,
// without making the core depend on that implementation.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}
