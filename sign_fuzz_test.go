//go:build go1.18
// +build go1.18

package mars

import (
	"testing"
)

func FuzzSignatureVerification(f *testing.F) {
	secretKey = generateRandomSecretKey()
	f.Add("4UcW-3rLvaGGxmA2KUPQgS30MVK7ESKKEPhs4Gir_-E")
	f.Fuzz(func(t *testing.T, sig string) {
		Verify("Untouchable", sig)
	})
}
