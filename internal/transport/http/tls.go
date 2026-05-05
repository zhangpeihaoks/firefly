// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSCfg is the TLS configuration for HTTP server.
type TLSCfg struct {
	// CertFile is the path to the certificate file.
	CertFile string
	// KeyFile is the path to the private key file.
	KeyFile string
	// CAFile is the path to the CA certificate file for client authentication.
	CAFile string
	// MinVersion is the minimum TLS version (e.g., tls.VersionTLS12).
	MinVersion uint16
	// MaxVersion is the maximum TLS version.
	MaxVersion uint16
	// PreferServerCipherSuites indicates whether to prefer server cipher suites.
	PreferServerCipherSuites bool
}

// LoadTLSConfig creates a tls.Config from the given TLS configuration.
// It loads certificates from files and configures TLS settings.
func LoadTLSCfg(cfg *TLSCfg) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}

	// If no certificate files specified, return nil (HTTP mode)
	if cfg.CertFile == "" && cfg.KeyFile == "" {
		return nil, nil
	}

	// Check if both certificate and key files are provided
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf("both cert_file and key_file are required for TLS")
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Build TLS config
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Load CA certificate for client authentication if provided
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConf.ClientCAs = caCertPool
		tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
	}

	// Set TLS version limits
	if cfg.MinVersion > 0 {
		tlsConf.MinVersion = cfg.MinVersion
	} else {
		tlsConf.MinVersion = tls.VersionTLS12
	}

	if cfg.MaxVersion > 0 {
		tlsConf.MaxVersion = cfg.MaxVersion
	}

	// Set cipher suite preference
	tlsConf.PreferServerCipherSuites = cfg.PreferServerCipherSuites

	return tlsConf, nil
}

// TLSOption is a function that configures the TLS configuration.
type TLSOption func(*TLSCfg)

// WithCertFile sets the certificate file path.
func WithCertFile(certFile string) TLSOption {
	return func(c *TLSCfg) {
		c.CertFile = certFile
	}
}

// WithKeyFile sets the private key file path.
func WithKeyFile(keyFile string) TLSOption {
	return func(c *TLSCfg) {
		c.KeyFile = keyFile
	}
}

// WithCAFile sets the CA certificate file path for client authentication.
func WithCAFile(caFile string) TLSOption {
	return func(c *TLSCfg) {
		c.CAFile = caFile
	}
}

// WithMinTLSVersion sets the minimum TLS version.
func WithMinTLSVersion(version uint16) TLSOption {
	return func(c *TLSCfg) {
		c.MinVersion = version
	}
}

// WithMaxTLSVersion sets the maximum TLS version.
func WithMaxTLSVersion(version uint16) TLSOption {
	return func(c *TLSCfg) {
		c.MaxVersion = version
	}
}

// WithPreferServerCipherSuites sets whether to prefer server cipher suites.
func WithPreferServerCipherSuites(prefer bool) TLSOption {
	return func(c *TLSCfg) {
		c.PreferServerCipherSuites = prefer
	}
}

// NewTLSCfg creates a new TLS configuration with the given options.
func NewTLSCfg(opts ...TLSOption) *TLSCfg {
	cfg := &TLSCfg{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
