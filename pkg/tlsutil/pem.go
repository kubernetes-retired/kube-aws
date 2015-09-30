package tlsutil

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
)

func WritePrivateKeyPEMBlock(out io.Writer, key *rsa.PrivateKey) error {
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.Encode(out, &block)
}

func WriteCertificatePEMBlock(out io.Writer, cert *x509.Certificate) error {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.Encode(out, &block)
}
