package config

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/tlsutil"
)

// PEM encoded TLS assets.
type RawTLSAssets struct {
	CACert        []byte
	CAKey         []byte
	APIServerCert []byte
	APIServerKey  []byte
	WorkerCert    []byte
	WorkerKey     []byte
	AdminCert     []byte
	AdminKey      []byte
}

// PEM -> gzip -> base64 encoded TLS assets.
type CompactTLSAssets struct {
	CACert        string
	CAKey         string
	APIServerCert string
	APIServerKey  string
	WorkerCert    string
	WorkerKey     string
	AdminCert     string
	AdminKey      string
}

func (c *Cluster) NewTLSAssets() (*RawTLSAssets, error) {
	// Convert from days to time.Duration
	caDuration := time.Duration(c.TLSCADurationDays) * 24 * time.Hour
	certDuration := time.Duration(c.TLSCertDurationDays) * 24 * time.Hour

	// Generate keys for the various components.
	keys := make([]*rsa.PrivateKey, 4)
	var err error
	for i := range keys {
		if keys[i], err = tlsutil.NewPrivateKey(); err != nil {
			return nil, err
		}
	}
	caKey, apiServerKey, workerKey, adminKey := keys[0], keys[1], keys[2], keys[3]

	caConfig := tlsutil.CACertConfig{
		CommonName:   "kube-ca",
		Organization: "kube-aws",
		Duration:     caDuration,
	}
	caCert, err := tlsutil.NewSelfSignedCACertificate(caConfig, caKey)
	if err != nil {
		return nil, err
	}

	//Compute kubernetesServiceIP from serviceCIDR
	_, serviceNet, err := net.ParseCIDR(c.ServiceCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid serviceCIDR: %v", err)
	}
	kubernetesServiceIPAddr := incrementIP(serviceNet.IP)

	apiServerConfig := tlsutil.ServerCertConfig{
		CommonName: "kube-apiserver",
		DNSNames: []string{
			"kubernetes",
			"kubernetes.default",
			"kubernetes.default.svc",
			"kubernetes.default.svc.cluster.local",
			c.ExternalDNSName,
		},
		IPAddresses: []string{
			c.ControllerIP,
			kubernetesServiceIPAddr.String(),
		},
		Duration: certDuration,
	}
	apiServerCert, err := tlsutil.NewSignedServerCertificate(apiServerConfig, apiServerKey, caCert, caKey)
	if err != nil {
		return nil, err
	}

	workerConfig := tlsutil.ClientCertConfig{
		CommonName: "kube-worker",
		DNSNames: []string{
			"*.*.compute.internal",
			"*.ec2.internal",
		},
		Duration: certDuration,
	}
	workerCert, err := tlsutil.NewSignedClientCertificate(workerConfig, workerKey, caCert, caKey)
	if err != nil {
		return nil, err
	}

	adminConfig := tlsutil.ClientCertConfig{
		CommonName: "kube-admin",
		Duration:   certDuration,
	}
	adminCert, err := tlsutil.NewSignedClientCertificate(adminConfig, adminKey, caCert, caKey)
	if err != nil {
		return nil, err
	}

	return &RawTLSAssets{
		CACert:        tlsutil.EncodeCertificatePEM(caCert),
		APIServerCert: tlsutil.EncodeCertificatePEM(apiServerCert),
		WorkerCert:    tlsutil.EncodeCertificatePEM(workerCert),
		AdminCert:     tlsutil.EncodeCertificatePEM(adminCert),
		CAKey:         tlsutil.EncodePrivateKeyPEM(caKey),
		APIServerKey:  tlsutil.EncodePrivateKeyPEM(apiServerKey),
		WorkerKey:     tlsutil.EncodePrivateKeyPEM(workerKey),
		AdminKey:      tlsutil.EncodePrivateKeyPEM(adminKey),
	}, nil
}

func ReadTLSAssets(dirname string) (*RawTLSAssets, error) {
	r := new(RawTLSAssets)
	files := []struct {
		name      string
		cert, key *[]byte
	}{
		{"ca", &r.CACert, &r.CAKey},
		{"apiserver", &r.APIServerCert, &r.APIServerKey},
		{"worker", &r.WorkerCert, &r.WorkerKey},
		{"admin", &r.AdminCert, &r.AdminKey},
	}
	for _, file := range files {
		certPath := filepath.Join(dirname, file.name+".pem")
		keyPath := filepath.Join(dirname, file.name+"-key.pem")

		certData, err := ioutil.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		*file.cert = certData
		keyData, err := ioutil.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		*file.key = keyData
	}
	return r, nil
}

func (r *RawTLSAssets) WriteToDir(dirname string) error {
	assets := []struct {
		name      string
		cert, key []byte
	}{
		{"ca", r.CACert, r.CAKey},
		{"apiserver", r.APIServerCert, r.APIServerKey},
		{"worker", r.WorkerCert, r.WorkerKey},
		{"admin", r.AdminCert, r.AdminKey},
	}
	for _, asset := range assets {
		certPath := filepath.Join(dirname, asset.name+".pem")
		keyPath := filepath.Join(dirname, asset.name+"-key.pem")
		if err := ioutil.WriteFile(certPath, asset.cert, 0600); err != nil {
			return err
		}
		if err := ioutil.WriteFile(keyPath, asset.key, 0600); err != nil {
			return err
		}
	}
	return nil
}

func compressData(d []byte) (string, error) {
	var buff bytes.Buffer
	gzw := gzip.NewWriter(&buff)
	if _, err := gzw.Write(d); err != nil {
		return "", err
	}
	if err := gzw.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buff.Bytes()), nil
}

type encryptService interface {
	Encrypt(*kms.EncryptInput) (*kms.EncryptOutput, error)
}

func (r *RawTLSAssets) compact(cfg *Config, kmsSvc encryptService) (*CompactTLSAssets, error) {
	var err error
	compact := func(data []byte) string {
		if err != nil {
			return ""
		}

		encryptInput := kms.EncryptInput{
			KeyId:     aws.String(cfg.KMSKeyARN),
			Plaintext: data,
		}

		var encryptOutput *kms.EncryptOutput
		if encryptOutput, err = kmsSvc.Encrypt(&encryptInput); err != nil {
			return ""
		}
		data = encryptOutput.CiphertextBlob

		var out string
		if out, err = compressData(data); err != nil {
			return ""
		}
		return out
	}
	compactAssets := CompactTLSAssets{
		CACert:        compact(r.CACert),
		CAKey:         compact(r.CAKey),
		APIServerCert: compact(r.APIServerCert),
		APIServerKey:  compact(r.APIServerKey),
		WorkerCert:    compact(r.WorkerCert),
		WorkerKey:     compact(r.WorkerKey),
		AdminCert:     compact(r.AdminCert),
		AdminKey:      compact(r.AdminKey),
	}
	if err != nil {
		return nil, err
	}
	return &compactAssets, nil
}
