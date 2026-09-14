package tlsutil_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jasonmiller-cc/parameters-core/pkg/tlsutil"
)

// generateSelfSignedCert writes a self-signed cert/key pair to dir and
// returns their paths. notAfter controls the certificate's expiry.
func generateSelfSignedCert(t *testing.T, dir string, notAfter time.Time) (certPath, keyPath string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test.parameters.cc"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:         true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certPath = filepath.Join(dir, "cert.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("encode cert: %v", err)
	}
	_ = certOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPath = filepath.Join(dir, "key.pem")
	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatalf("encode key: %v", err)
	}
	_ = keyOut.Close()

	return certPath, keyPath
}

func TestLoadCertPool(t *testing.T) {
	dir := t.TempDir()
	certPath, _ := generateSelfSignedCert(t, dir, time.Now().Add(24*time.Hour))

	pool, err := tlsutil.LoadCertPool(certPath)
	if err != nil {
		t.Fatalf("LoadCertPool() error: %v", err)
	}
	if pool == nil {
		t.Fatal("LoadCertPool() returned nil pool")
	}
}

func TestLoadCertPool_MissingFile(t *testing.T) {
	if _, err := tlsutil.LoadCertPool("/nonexistent/ca.pem"); err == nil {
		t.Fatal("expected error for missing CA file, got nil")
	}
}

func TestLoadCertPool_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := tlsutil.LoadCertPool(path); err == nil {
		t.Fatal("expected error for invalid PEM, got nil")
	}
}

func TestLoadKeyPair(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir, time.Now().Add(24*time.Hour))

	cert, err := tlsutil.LoadKeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("LoadKeyPair() error: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Error("LoadKeyPair() returned empty certificate chain")
	}
}

func TestExpiresIn(t *testing.T) {
	dir := t.TempDir()
	notAfter := time.Now().Add(48 * time.Hour)
	certPath, _ := generateSelfSignedCert(t, dir, notAfter)

	got, err := tlsutil.ExpiresIn(certPath)
	if err != nil {
		t.Fatalf("ExpiresIn() error: %v", err)
	}
	// Allow a small tolerance for test execution time.
	want := 48 * time.Hour
	diff := want - got
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Minute {
		t.Errorf("ExpiresIn() = %v, want ~%v", got, want)
	}
}

func TestExpiresIn_MissingFile(t *testing.T) {
	if _, err := tlsutil.ExpiresIn("/nonexistent/cert.pem"); err == nil {
		t.Fatal("expected error for missing cert file, got nil")
	}
}

func TestExpiresIn_NoPEMBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.pem")
	if err := os.WriteFile(path, []byte("not pem data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := tlsutil.ExpiresIn(path); err == nil {
		t.Fatal("expected error for non-PEM file, got nil")
	}
}

func TestServerTLSConfig(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir, time.Now().Add(24*time.Hour))

	cfg, err := tlsutil.ServerTLSConfig(certPath, keyPath, "")
	if err != nil {
		t.Fatalf("ServerTLSConfig() error: %v", err)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("Certificates = %d, want 1", len(cfg.Certificates))
	}
}

func TestServerTLSConfig_WithClientCA(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir, time.Now().Add(24*time.Hour))

	cfg, err := tlsutil.ServerTLSConfig(certPath, keyPath, certPath)
	if err != nil {
		t.Fatalf("ServerTLSConfig() error: %v", err)
	}
	if cfg.ClientCAs == nil {
		t.Error("expected ClientCAs to be set when caFile is provided")
	}
}

func TestClientTLSConfig(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir, time.Now().Add(24*time.Hour))

	cfg, err := tlsutil.ClientTLSConfig(certPath, keyPath, certPath)
	if err != nil {
		t.Fatalf("ClientTLSConfig() error: %v", err)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("Certificates = %d, want 1", len(cfg.Certificates))
	}
	if cfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
}

func TestClientTLSConfig_NoCert(t *testing.T) {
	cfg, err := tlsutil.ClientTLSConfig("", "", "")
	if err != nil {
		t.Fatalf("ClientTLSConfig() error: %v", err)
	}
	if len(cfg.Certificates) != 0 {
		t.Error("expected no certificates when certFile/keyFile are empty")
	}
}
