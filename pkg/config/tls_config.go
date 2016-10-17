package config

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"crypto/x509"
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
	CACert         []byte
	CAKey          []byte
	APIServerCert  []byte
	APIServerKey   []byte
	WorkerCert     []byte
	WorkerKey      []byte
	AdminCert      []byte
	AdminKey       []byte
	EtcdCert       []byte
	EtcdClientCert []byte
	EtcdKey        []byte
	EtcdClientKey  []byte
}

// PEM -> gzip -> base64 encoded TLS assets.
type CompactTLSAssets struct {
	CACert         string
	CAKey          string
	APIServerCert  string
	APIServerKey   string
	WorkerCert     string
	WorkerKey      string
	AdminCert      string
	AdminKey       string
	EtcdCert       string
	EtcdClientCert string
	EtcdClientKey  string
	EtcdKey        string
}

func NewTLSCA() (*rsa.PrivateKey, *x509.Certificate, error) {
	caKey, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	caConfig := tlsutil.CACertConfig{
		CommonName:   "kube-ca",
		Organization: "kube-aws",
	}
	caCert, err := tlsutil.NewSelfSignedCACertificate(caConfig, caKey)
	if err != nil {
		return nil, nil, err
	}

	return caKey, caCert, nil
}

func (c *Cluster) NewTLSAssets(caKey *rsa.PrivateKey, caCert *x509.Certificate) (*RawTLSAssets, error) {
	// Convert from days to time.Duration
	certDuration := time.Duration(c.TLSCertDurationDays) * 24 * time.Hour

	// Generate keys for the various components.
	keys := make([]*rsa.PrivateKey, 5)
	var err error
	for i := range keys {
		if keys[i], err = tlsutil.NewPrivateKey(); err != nil {
			return nil, err
		}
	}
	apiServerKey, workerKey, adminKey, etcdKey, etcdClientKey := keys[0], keys[1], keys[2], keys[3], keys[4]

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
			kubernetesServiceIPAddr.String(),
		},
		Duration: certDuration,
	}
	apiServerCert, err := tlsutil.NewSignedServerCertificate(apiServerConfig, apiServerKey, caCert, caKey)
	if err != nil {
		return nil, err
	}

	etcdConfig := tlsutil.ServerCertConfig{
		CommonName: "kube-etcd",
		DNSNames: []string{
			fmt.Sprintf("*.%s.compute.internal", c.Region),
			"*.ec2.internal",
		},
		//etcd https client/peer interfaces are not exposed externally
		//will live the full year with the CA
		Duration: tlsutil.Duration365d,
	}

	etcdCert, err := tlsutil.NewSignedServerCertificate(etcdConfig, etcdKey, caCert, caKey)
	if err != nil {
		return nil, err
	}

	workerConfig := tlsutil.ClientCertConfig{
		CommonName: "kube-worker",
		DNSNames: []string{
			fmt.Sprintf("*.%s.compute.internal", c.Region),
			"*.ec2.internal",
		},
		Duration: certDuration,
	}
	workerCert, err := tlsutil.NewSignedClientCertificate(workerConfig, workerKey, caCert, caKey)
	if err != nil {
		return nil, err
	}

	etcdClientConfig := tlsutil.ClientCertConfig{
		CommonName: "kube-etcd-client",
	}

	etcdClientCert, err := tlsutil.NewSignedClientCertificate(etcdClientConfig, etcdClientKey, caCert, caKey)
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
		CACert:         tlsutil.EncodeCertificatePEM(caCert),
		APIServerCert:  tlsutil.EncodeCertificatePEM(apiServerCert),
		WorkerCert:     tlsutil.EncodeCertificatePEM(workerCert),
		AdminCert:      tlsutil.EncodeCertificatePEM(adminCert),
		EtcdCert:       tlsutil.EncodeCertificatePEM(etcdCert),
		EtcdClientCert: tlsutil.EncodeCertificatePEM(etcdClientCert),
		CAKey:          tlsutil.EncodePrivateKeyPEM(caKey),
		APIServerKey:   tlsutil.EncodePrivateKeyPEM(apiServerKey),
		WorkerKey:      tlsutil.EncodePrivateKeyPEM(workerKey),
		AdminKey:       tlsutil.EncodePrivateKeyPEM(adminKey),
		EtcdKey:        tlsutil.EncodePrivateKeyPEM(etcdKey),
		EtcdClientKey:  tlsutil.EncodePrivateKeyPEM(etcdClientKey),
	}, nil
}

func ReadTLSAssets(dirname string) (*RawTLSAssets, error) {
	r := new(RawTLSAssets)
	files := []struct {
		name      string
		cert, key *[]byte
	}{
		{"ca", &r.CACert, nil},
		{"apiserver", &r.APIServerCert, &r.APIServerKey},
		{"worker", &r.WorkerCert, &r.WorkerKey},
		{"admin", &r.AdminCert, &r.AdminKey},
		{"etcd", &r.EtcdCert, &r.EtcdKey},
		{"etcd-client", &r.EtcdClientCert, &r.EtcdClientKey},
	}
	for _, file := range files {
		certPath := filepath.Join(dirname, file.name+".pem")
		keyPath := filepath.Join(dirname, file.name+"-key.pem")

		certData, err := ioutil.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		*file.cert = certData

		if file.key != nil {
			keyData, err := ioutil.ReadFile(keyPath)
			if err != nil {
				return nil, err
			}
			*file.key = keyData
		}
	}
	return r, nil
}

func (r *RawTLSAssets) WriteToDir(dirname string, includeCAKey bool) error {
	assets := []struct {
		name      string
		cert, key []byte
	}{
		{"ca", r.CACert, r.CAKey},
		{"apiserver", r.APIServerCert, r.APIServerKey},
		{"worker", r.WorkerCert, r.WorkerKey},
		{"admin", r.AdminCert, r.AdminKey},
		{"etcd", r.EtcdCert, r.EtcdKey},
		{"etcd-client", r.EtcdClientCert, r.EtcdClientKey},
	}
	for _, asset := range assets {
		certPath := filepath.Join(dirname, asset.name+".pem")
		keyPath := filepath.Join(dirname, asset.name+"-key.pem")

		if err := ioutil.WriteFile(certPath, asset.cert, 0600); err != nil {
			return err
		}

		if asset.name != "ca" || includeCAKey {
			if err := ioutil.WriteFile(keyPath, asset.key, 0600); err != nil {
				return err
			}
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
		CACert:         compact(r.CACert),
		APIServerCert:  compact(r.APIServerCert),
		APIServerKey:   compact(r.APIServerKey),
		WorkerCert:     compact(r.WorkerCert),
		WorkerKey:      compact(r.WorkerKey),
		AdminCert:      compact(r.AdminCert),
		AdminKey:       compact(r.AdminKey),
		EtcdCert:       compact(r.EtcdCert),
		EtcdClientCert: compact(r.EtcdClientCert),
		EtcdClientKey:  compact(r.EtcdClientKey),
		EtcdKey:        compact(r.EtcdKey),
	}
	if err != nil {
		return nil, err
	}
	return &compactAssets, nil
}
