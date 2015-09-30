package tlsutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math"
	"math/big"
	"net"
	"time"
)

var (
	Duration90d  = time.Hour * 24 * 90
	Duration365d = time.Hour * 24 * 365
)

type CACertConfig struct {
	CommonName   string
	Organization string
}

type ServerCertConfig struct {
	CommonName  string
	DNSNames    []string
	IPAddresses []string
}

type ClientCertConfig struct {
	CommonName  string
	DNSNames    []string
	IPAddresses []string
}

func NewSelfSignedCACertificate(cfg CACertConfig, key *rsa.PrivateKey) (*x509.Certificate, error) {
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: []string{cfg.Organization},
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(Duration365d).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA: true,
	}

	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func NewSignedServerCertificate(cfg ServerCertConfig, key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, error) {
	ips := make([]net.IP, len(cfg.IPAddresses))
	for i, ipStr := range cfg.IPAddresses {
		ips[i] = net.ParseIP(ipStr)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: caCert.Subject.Organization,
		},
		DNSNames:     cfg.DNSNames,
		IPAddresses:  ips,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(Duration90d).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func NewSignedClientCertificate(cfg ClientCertConfig, key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, error) {
	ips := make([]net.IP, len(cfg.IPAddresses))
	for i, ipStr := range cfg.IPAddresses {
		ips[i] = net.ParseIP(ipStr)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: caCert.Subject.Organization,
		},
		DNSNames:     cfg.DNSNames,
		IPAddresses:  ips,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(Duration90d).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}
