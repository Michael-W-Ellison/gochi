package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestShippedConfigMatchesDefaults(t *testing.T) {
	config, err := loadConfig(filepath.Join("..", "..", "configs", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if config != defaultConfig() {
		t.Errorf("configs/config.yaml should hold the defaults:\n got %+v\nwant %+v", config, defaultConfig())
	}
}

func TestLoadConfig(t *testing.T) {
	path := writeConfig(t, `
server:
  addr: ":9090"
  data_dir: "saves-here"
simulation:
  time_scale: "24x"
  autosave: 30s
`)
	config, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Server.Addr != ":9090" || config.Simulation.TimeScale != "24x" || config.Simulation.Autosave != 30*time.Second {
		t.Errorf("Settings not read: %+v", config)
	}
	if config.Simulation.AwayPause != 2*time.Minute {
		t.Errorf("A setting left out should keep its default, got %s", config.Simulation.AwayPause)
	}
	if want := filepath.Join(filepath.Dir(path), "saves-here"); config.Server.DataDir != want {
		t.Errorf("A relative data_dir should be relative to the file: got %s, want %s", config.Server.DataDir, want)
	}

	if config, err := loadConfig(writeConfig(t, "")); err != nil || config != defaultConfig() {
		t.Errorf("An empty file should give the defaults: %+v, %v", config, err)
	}
}

func TestLoadConfigRejectsBadSettings(t *testing.T) {
	for text, want := range map[string]string{
		"simulation:\n  tick_rate: 60\n":        `unknown setting "tick_rate"`,
		"simulation:\n  time_scale: \"fast\"\n": "unknown time_scale",
		"simulation:\n  autosave: -1m\n":        "can't be negative",
		"simulation:\n  autosave: often\n":      "often",
		"server:\n  api_rate_limit: -5\n":       "api_rate_limit can't be negative",
	} {
		if _, err := loadConfig(writeConfig(t, text)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: expected an error mentioning %q, got %v", text, want, err)
		}
	}
	if _, err := loadConfig(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("A missing file should be an error")
	}
}

// writeCert writes a self-signed certificate and key for localhost to dir
func writeCert(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"},
		DNSNames: []string{"localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certFile, keyFile = filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

func TestTLSSettings(t *testing.T) {
	dir := t.TempDir()
	writeCert(t, dir)
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  tls_cert: cert.pem\n  tls_key: key.pem\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !config.usesTLS() || config.Server.TLSCert != filepath.Join(dir, "cert.pem") {
		t.Errorf("Relative certificate paths are relative to the config file: %+v", config.Server)
	}

	if _, err := loadConfig(writeConfig(t, "server:\n  tls_cert: cert.pem\n")); err == nil || !strings.Contains(err.Error(), "both") {
		t.Errorf("A certificate without a key: %v", err)
	}
	if _, err := loadConfig(writeConfig(t, "server:\n  tls_cert: missing.pem\n  tls_key: missing.pem\n")); err == nil {
		t.Error("A missing certificate should be an error")
	}
	if defaultConfig().usesTLS() {
		t.Error("HTTPS is off by default")
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	securityHeaders(http.NotFoundHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("Headers: %v", rec.Header())
	}
}
