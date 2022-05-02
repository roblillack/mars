package mars

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

var HashAlgorithm = sha256.New

// Sign a given string with the app-configured secret key.
// If no secret key is set, returns the empty string.
// Return the signature in base64 (URLEncoding).
func Sign(message string) string {
	if len(secretKey) == 0 {
		return ""
	}
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
