package ch

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

	"github.com/stretchr/testify/require"
)

// writeTestCerts generates a self-signed leaf cert + key plus a CA cert PEM
// in dir and returns their paths. The CA file in this test setup is the same
// self-signed certificate (the simplest valid PEM with at least one cert).
func writeTestCerts(t *testing.T, dir string) (caPath, certPath, keyPath string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "chtop-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	caPath = filepath.Join(dir, "ca.pem")
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	require.NoError(t, os.WriteFile(caPath,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0o600))
	require.NoError(t, os.WriteFile(certPath,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0o600))
	require.NoError(t, os.WriteFile(keyPath,
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))
	return caPath, certPath, keyPath
}

func TestBuildTLSConfig(t *testing.T) {
	t.Parallel()

	type assertion func(t *testing.T, tlsCfg *anyTLSConfig, err error)

	// We swap *tls.Config for anyTLSConfig in the assert signature only so
	// the test file doesn't import crypto/tls just to read field types — the
	// real type comes from the helper, and we expose its key parts here.

	dir := t.TempDir()
	caPath, certPath, keyPath := writeTestCerts(t, dir)
	missing := filepath.Join(dir, "does-not-exist.pem")

	type testCase struct {
		name   string
		cfg    Config
		assert assertion
	}
	cases := []testCase{
		{
			name: "TLS disabled returns nil with no error",
			cfg:  Config{Host: "x"},
			assert: func(t *testing.T, c *anyTLSConfig, err error) {
				require.NoError(t, err)
				require.Nil(t, c)
			},
		},
		{
			name: "TLS enabled bare returns empty config",
			cfg:  Config{Host: "x", TLS: true},
			assert: func(t *testing.T, c *anyTLSConfig, err error) {
				require.NoError(t, err)
				require.NotNil(t, c)
				require.Nil(t, c.RootCAs)
				require.Empty(t, c.Certificates)
			},
		},
		{
			name: "with ca file populates RootCAs",
			cfg:  Config{Host: "x", TLS: true, TLSCAFile: caPath},
			assert: func(t *testing.T, c *anyTLSConfig, err error) {
				require.NoError(t, err)
				require.NotNil(t, c)
				require.NotNil(t, c.RootCAs)
			},
		},
		{
			name: "with cert + key populates Certificates",
			cfg: Config{
				Host: "x", TLS: true,
				TLSCertFile: certPath, TLSKeyFile: keyPath,
			},
			assert: func(t *testing.T, c *anyTLSConfig, err error) {
				require.NoError(t, err)
				require.NotNil(t, c)
				require.Len(t, c.Certificates, 1)
			},
		},
		{
			name: "missing ca file returns error",
			cfg:  Config{Host: "x", TLS: true, TLSCAFile: missing},
			assert: func(t *testing.T, c *anyTLSConfig, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "cert without key is rejected",
			cfg: Config{
				Host: "x", TLS: true, TLSCertFile: certPath,
			},
			assert: func(t *testing.T, c *anyTLSConfig, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := buildTLSConfig(tc.cfg)
			tc.assert(t, wrapTLS(got), err)
		})
	}
}
