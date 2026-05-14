package ch

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// buildTLSConfig converts the TLS-related fields of Config into a *tls.Config.
// Pure: depends only on its input and the contents of the files it points at.
//
//   - TLS=false              -> nil, nil
//   - TLS=true, nothing else -> empty &tls.Config{}, nil
//   - TLSCAFile set          -> RootCAs populated from the PEM file
//   - TLSCertFile+TLSKeyFile -> Certificates loaded from disk (both must be
//     set together — half a key pair is a user error)
func buildTLSConfig(cfg Config) (*tls.Config, error) {
	if !cfg.TLS {
		return nil, nil
	}
	out := &tls.Config{}
	if cfg.TLSCAFile != "" {
		pemBytes, err := os.ReadFile(cfg.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("read tls ca %s: %w", cfg.TLSCAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemBytes) {
			return nil, fmt.Errorf("tls ca %s: no certificates parsed", cfg.TLSCAFile)
		}
		out.RootCAs = pool
	}
	if cfg.TLSCertFile != "" || cfg.TLSKeyFile != "" {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			return nil, errors.New("tls cert and key must both be set or both empty")
		}
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load tls keypair: %w", err)
		}
		out.Certificates = []tls.Certificate{cert}
	}
	return out, nil
}

// anyTLSConfig and wrapTLS exist only for the test file to peek at the
// returned config without importing crypto/tls there. They are intentionally
// unexported.
type anyTLSConfig struct {
	RootCAs      *x509.CertPool
	Certificates []tls.Certificate
}

func wrapTLS(c *tls.Config) *anyTLSConfig {
	if c == nil {
		return nil
	}
	return &anyTLSConfig{RootCAs: c.RootCAs, Certificates: c.Certificates}
}
