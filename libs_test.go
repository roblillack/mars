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
