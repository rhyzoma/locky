package security

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/rhyzoma/locky/internal/config"
)

func ServerTLSConfig(cfg config.TLSConfig) (*tls.Config, error) {
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, nil
	}
	certificate, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server certificate: %w", err)
	}
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{certificate},
	}
	if cfg.ClientCAFile == "" {
		return tlsConfig, nil
	}
	data, err := os.ReadFile(cfg.ClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read client ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, errors.New("load client ca: no certificates found")
	}
	tlsConfig.ClientCAs = pool
	if cfg.RequireClientCert {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return tlsConfig, nil
}

func ClientTLSConfig(cfg config.TLSConfig) (*tls.Config, error) {
	if cfg.ClientCAFile == "" && cfg.CertFile == "" && cfg.KeyFile == "" {
		return nil, nil
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13}
	if cfg.ClientCAFile != "" {
		data, err := os.ReadFile(cfg.ClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("read server ca: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(data) {
			return nil, errors.New("load server ca: no certificates found")
		}
		tlsConfig.RootCAs = pool
	}
	if cfg.CertFile == "" && cfg.KeyFile == "" {
		return tlsConfig, nil
	}
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, errors.New("client certificate and key must be provided together")
	}
	certificate, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load client certificate: %w", err)
	}
	tlsConfig.Certificates = []tls.Certificate{certificate}
	return tlsConfig, nil
}
