package mars

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

var HashAlgorithm = sha256.New
var HashBlockSize = sha256.BlockSize

var (
	// Private
	secretKey []byte // Key used to sign cookies.
)

func SetAppSecret(secret string) {
	secretKey = []byte(secret)
}

func generateRandomSecretKey() []byte {
	buf := make([]byte, HashBlockSize)
	if _, err := rand.Read(buf); err != nil {
		panic("Unable to generate random application secret")
	}

	return buf
}

// Sign a given string with the configured or random secret key.
// If no secret key is set, returns the empty string.
// Return the signature in unpadded, URL-safe base64 encoding
// (A-Z, 0-9, a-z, _ and -).
func Sign(message string) string {
	mac := hmac.New(HashAlgorithm, secretKey)
	io.WriteString(mac, message)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Verify returns true if the given signature is correct for the given message.
// e.g. it matches what we generate with Sign()
func Verify(message, sig string) bool {
	// return hmac.Equal([]byte(sig), []byte(Sign(message)))
	mac := hmac.New(HashAlgorithm, secretKey)
	io.WriteString(mac, message)
	hash := mac.Sum(nil)

	received, _ := base64.RawURLEncoding.DecodeString(sig)
	return hmac.Equal(received, hash)
}
