package mars

import (
	"fmt"
	"testing"
)

func BenchmarkSigning(b *testing.B) {
	SetAppSecret("Ludolfs lustige Liegestütze ließen Lolas Lachmuskeln leuchten.")

	for n := 0; n < b.N; n++ {
		str := fmt.Sprintf("%d", n)
		sig := Sign(str)
		if ok := Verify(str, sig); !ok {
			b.Fatalf("signature '%s' of '%s' cannot be verified!", sig, str)
		}
	}
}

func TestSimpleSignatures(t *testing.T) {
	SetAppSecret("Kurts käsiger Kugelbauch konterte Karins kichernden Kuss.")

	for msg, sig := range map[string]string{
		"Untouchable": "4UcW-3rLvaGGxmA2KUPQgS30MVK7ESKKEPhs4Gir_-E",
		"///a/a///a/": "vYMQQF_m2JnfKa5l0aBt1Iub_IhTu0ZWRcTWDC-oaxE",
	} {
		if r := Sign(msg); r != sig {
			t.Fatalf("wrong signature '%s' for '%s'!", r, msg)

		}
		if !Verify(msg, sig) {
			t.Fatalf("signature '%s' of '%s' cannot be verified!", sig, msg)

		}
	}
}

func TestSignature(t *testing.T) {
	SetAppSecret("Richards rüstige Rottweilerdame Renate riss ruchloserweise Rehe.")

	for n := 0; n < 100; n++ {
		msg := "Untouchable " + generateRandomToken()
		sig := Sign(msg)
		if len(sig) != 43 {
			t.Fatalf("wrong signature length %d for '%s' (sig: '%s')!", len(sig), msg, sig)

		}
		if !Verify(msg, sig) {
			t.Fatalf("signature '%s' of '%s' cannot be verified!", sig, msg)

		}
	}
}

func TestSignatureWithRandomSecret(t *testing.T) {
	for i := 0; i < 100; i++ {
		secretKey = generateRandomSecretKey()
		for n := 0; n < 100; n++ {
			msg := "Untouchable " + generateRandomToken()
			sig := Sign(msg)
			if len(sig) != 43 {
				t.Fatalf("wrong signature length %d for '%s' (sig: '%s')!", len(sig), msg, sig)

			}
			if !Verify(msg, sig) {
				t.Fatalf("signature '%s' of '%s' cannot be verified!", sig, msg)

			}
		}
	}
}
