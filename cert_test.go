package mars

import (
	"crypto/x509"
	"strings"
	"testing"
)

func TestCertificateCreation(t *testing.T) {
	for org, domains := range map[string][]string{
		"ACME Inc.": []string{"acme.com", "acme.biz"},
		"Me":        []string{"::1", "127.0.0.1"},
	} {
		keypair, err := createCertificate(org, strings.Join(domains, ", "))
		if err != nil {
			t.Fatal(err)
		}

		cert, err := x509.ParseCertificate(keypair.Certificate[0])
		if err != nil {
			t.Fatal(err)
		}

		for _, i := range domains {
			if err := cert.VerifyHostname(i); err != nil {
				t.Errorf("Unable to validate host %s for %s: %s", i, org, err)
			}
		}
	}
}
