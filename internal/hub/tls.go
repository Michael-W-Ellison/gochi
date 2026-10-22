package hub

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Hubs always use HTTPS. A hub without a certificate from a certificate
// authority makes its own (self-signed) and keeps it; a game connecting to
// such a hub remembers the certificate's fingerprint when it first
// connects and refuses any other certificate afterwards (pinning), so
// nobody can stand in for the hub later. The hub prints the fingerprint at
// startup so the player's parent can compare it with the one the game
// shows. Plain HTTP is accepted only for a hub on the same computer.

// Self-signed certificate files in the hub's directory
const (
	certFileName = "hub-cert.pem"
	keyFileName  = "hub-key.pem"
)

// EnsureCertificate returns the hub's self-signed certificate and key in
// dir, making them if they don't exist yet
func EnsureCertificate(dir string) (certFile, keyFile string, err error) {
	certFile, keyFile = filepath.Join(dir, certFileName), filepath.Join(dir, keyFileName)
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
		return certFile, keyFile, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		return "", "", err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "Gochi hub"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	if host, err := os.Hostname(); err == nil && host != "" {
		template.DNSNames = append(template.DNSNames, host)
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return "", "", err
	}
	return certFile, keyFile, nil
}

// Fingerprint returns a certificate's SHA-256 fingerprint, as pairs of hex
// digits separated by colons
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = hex.EncodeToString([]byte{b})
	}
	return strings.ToUpper(strings.Join(parts, ":"))
}

// CertificateFingerprint returns the fingerprint of the certificate in a
// PEM file
func CertificateFingerprint(certFile string) (string, error) {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return "", errors.New("no certificate in " + certFile)
	}
	return Fingerprint(block.Bytes), nil
}

// SetTimeouts sets how long a hub server waits for requests: a request,
// including an upload, must arrive within a minute, so slow clients can't
// hold connections open
func SetTimeouts(server *http.Server) {
	server.ReadHeaderTimeout = 10 * time.Second
	server.ReadTimeout = time.Minute
	server.WriteTimeout = 2 * time.Minute
	server.IdleTimeout = 2 * time.Minute
}

// TLSConfig is the hub's TLS configuration: TLS 1.2 or later
func TLSConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

// loopbackHost reports whether a URL's host is this computer
func loopbackHost(u *url.URL) bool {
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// transport returns an HTTP transport that checks the hub's certificate:
// against pin if it is set, otherwise as usual
func transport(pin string) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if pin != "" {
		t.TLSClientConfig.InsecureSkipVerify = true // Replaced by the pin check
		t.TLSClientConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 || Fingerprint(cs.PeerCertificates[0].Raw) != pin {
				return ErrCertificateChanged
			}
			return nil
		}
	}
	return t
}

// ErrCertificateChanged is returned when a hub's certificate isn't the one
// remembered when the game connected
var ErrCertificateChanged = errors.New("the hub's certificate has changed since you connected; " +
	"this could be someone pretending to be the hub, so nothing was sent. If the hub's owner made a new " +
	"certificate, disconnect and connect again")

// ProbeCertificate connects to a hub and returns the pin to use for it:
// "" if its certificate is trusted by this computer (from a certificate
// authority), otherwise its fingerprint, to remember
func ProbeCertificate(ctx context.Context, baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "http" {
		return "", nil // Only allowed on this computer (NormalizeURL)
	}
	host := u.Host
	if u.Port() == "" {
		host = net.JoinHostPort(u.Hostname(), "443")
	}
	dialer := &tls.Dialer{Config: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return "", fmt.Errorf("could not reach the hub: %w", err)
	}
	defer conn.Close() //nolint:errcheck // Only the certificate is read
	state := conn.(*tls.Conn).ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return "", errors.New("the hub sent no certificate")
	}
	leaf := state.PeerCertificates[0]
	intermediates := x509.NewCertPool()
	for _, c := range state.PeerCertificates[1:] {
		intermediates.AddCert(c)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{DNSName: u.Hostname(), Intermediates: intermediates}); err == nil {
		return "", nil
	}
	return Fingerprint(leaf.Raw), nil
}

// ServerCertificate returns the certificate and key a hub serves (the
// given files, or if there are none its own self-signed certificate in
// dir) and the certificate's fingerprint
func ServerCertificate(dir, certFile, keyFile string) (string, string, string, error) {
	if (certFile == "") != (keyFile == "") {
		return "", "", "", errors.New("a certificate needs both its file and its key")
	}
	if certFile == "" {
		var err error
		if certFile, keyFile, err = EnsureCertificate(dir); err != nil {
			return "", "", "", err
		}
	}
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		return "", "", "", err
	}
	fingerprint, err := CertificateFingerprint(certFile)
	return certFile, keyFile, fingerprint, err
}
