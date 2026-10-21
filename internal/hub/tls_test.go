package hub

import (
	"crypto/tls"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

// tlsHub starts a hub over HTTPS with its own self-signed certificate
func tlsHub(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	h, err := New(dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	certFile, keyFile, err := EnsureCertificate(dir)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(h.Handler())
	server.TLS = TLSConfig()
	server.TLS.Certificates = []tls.Certificate{pair}
	server.StartTLS()
	t.Cleanup(server.Close)
	fingerprint, err := CertificateFingerprint(certFile)
	if err != nil {
		t.Fatal(err)
	}
	return server, fingerprint
}

func TestSelfSignedCertificateIsPinned(t *testing.T) {
	server, fingerprint := tlsHub(t)

	// The game learns the fingerprint when it connects
	pin, err := ProbeCertificate(ctx, server.URL)
	if err != nil || pin != fingerprint {
		t.Fatalf("Probe: %q %v, want %q", pin, err, fingerprint)
	}
	if len(strings.Split(pin, ":")) != 32 {
		t.Errorf("Fingerprint format: %q", pin)
	}
	if _, err := NewClient(server.URL, "", pin).Status(ctx); err != nil {
		t.Errorf("With the right pin: %v", err)
	}

	// Another certificate (someone standing in for the hub) is refused
	other, _ := tlsHub(t)
	if _, err := NewClient(other.URL, "", pin).Status(ctx); !errors.Is(err, ErrCertificateChanged) {
		t.Errorf("A different certificate should be refused: %v", err)
	}
	// Without a pin, a certificate this computer doesn't trust is refused
	if _, err := NewClient(server.URL, "", "").Status(ctx); err == nil {
		t.Error("An untrusted certificate without a pin should be refused")
	}
}

func TestCertificateIsKept(t *testing.T) {
	dir := t.TempDir()
	cert1, _, err := EnsureCertificate(dir)
	if err != nil {
		t.Fatal(err)
	}
	f1, _ := CertificateFingerprint(cert1)
	cert2, _, _ := EnsureCertificate(dir)
	if f2, _ := CertificateFingerprint(cert2); f1 != f2 {
		t.Error("The hub should keep its certificate, so pins stay valid")
	}
}
