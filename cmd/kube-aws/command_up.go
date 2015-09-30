package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/cluster"
	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/tlsutil"
)

var (
	cmdUp = &cobra.Command{
		Use:   "up",
		Short: "Create a new Kubernetes cluster",
		Long:  ``,
		Run:   runCmdUp,
	}

	kubeconfigTemplate *template.Template
)

func init() {
	kubeconfigTemplate = template.Must(template.New("kubeconfig").Parse(kubeconfigTemplateContents))

	cmdRoot.AddCommand(cmdUp)
}

func runCmdUp(cmd *cobra.Command, args []string) {
	cfg := cluster.NewDefaultConfig(VERSION)
	err := cluster.DecodeConfigFromFile(cfg, rootOpts.ConfigPath)
	if err != nil {
		stderr("Unable to load cluster config: %v", err)
		os.Exit(1)
	}

	c := cluster.New(cfg, newAWSConfig(cfg))

	clusterDir, err := filepath.Abs(path.Join("clusters", cfg.ClusterName))
	if err != nil {
		stderr("Unable to expand cluster directory to absolute path: %v", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(clusterDir, 0700); err != nil {
		stderr("Failed creating cluster workspace %s: %v", clusterDir, err)
		os.Exit(1)
	}

	tlsConfig, err := initTLS(cfg, clusterDir)
	if err != nil {
		stderr("Failed initializing TLS infrastructure: %v", err)
		os.Exit(1)
	}

	fmt.Println("Initialized TLS infrastructure")

	kubeconfig, err := newKubeconfig(cfg, tlsConfig)
	if err != nil {
		stderr("Failed rendering kubeconfig: %v", err)
		os.Exit(1)
	}

	kubeconfigPath := path.Join(clusterDir, "kubeconfig")
	if err := ioutil.WriteFile(kubeconfigPath, kubeconfig, 0600); err != nil {
		stderr("Failed writing kubeconfig to %s: %v", kubeconfigPath, err)
		os.Exit(1)
	}

	fmt.Printf("Wrote kubeconfig to %s\n", kubeconfigPath)

	fmt.Println("Waiting for cluster creation...")

	if err := c.Create(tlsConfig); err != nil {
		stderr("Failed creating cluster: %v", err)
		os.Exit(1)
	}

	fmt.Println("Successfully created cluster")
	fmt.Println("")

	info, err := c.Info()
	if err != nil {
		stderr("Failed fetching cluster info: %v", err)
		os.Exit(1)
	}

	fmt.Printf(info.String())
}

func getCloudFormation(url string) (string, error) {
	r, err := http.Get(url)

	if err != nil {
		return "", fmt.Errorf("Failed to get template: %v", err)
	}

	if r.StatusCode != 200 {
		return "", fmt.Errorf("Failed to get template: invalid status code: %d", r.StatusCode)
	}

	tmpl, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to get template: %v", err)
	}
	r.Body.Close()

	return string(tmpl), nil
}

func mustReadFile(loc string) []byte {
	b, err := ioutil.ReadFile(loc)
	if err != nil {
		stderr("Failed reading file %s: %s", loc, err)
		os.Exit(1)
	}
	return b
}

func newKubeconfig(cfg *cluster.Config, tlsConfig *cluster.TLSConfig) ([]byte, error) {
	data := struct {
		ClusterName       string
		APIServerEndpoint string
		AdminCertFile     string
		AdminKeyFile      string
		CACertFile        string
	}{
		ClusterName:       cfg.ClusterName,
		APIServerEndpoint: fmt.Sprintf("https://%s", cfg.ExternalDNSName),
		AdminCertFile:     tlsConfig.AdminCertFile,
		AdminKeyFile:      tlsConfig.AdminKeyFile,
		CACertFile:        tlsConfig.CACertFile,
	}

	var rendered bytes.Buffer
	if err := kubeconfigTemplate.Execute(&rendered, data); err != nil {
		return nil, err
	}

	return rendered.Bytes(), nil
}

var kubeconfigTemplateContents = `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority: {{ .CACertFile }}
    server: {{ .APIServerEndpoint }}
  name: kube-aws-{{ .ClusterName }}-cluster
contexts:
- context:
    cluster: kube-aws-{{ .ClusterName }}-cluster
    namespace: default
    user: kube-aws-{{ .ClusterName }}-admin
  name: kube-aws-{{ .ClusterName }}-context
users:
- name: kube-aws-{{ .ClusterName }}-admin
  user:
    client-certificate: {{ .AdminCertFile }}
    client-key: {{ .AdminKeyFile }}
current-context: kube-aws-{{ .ClusterName }}-context
`

func initTLS(cfg *cluster.Config, dir string) (*cluster.TLSConfig, error) {
	caCertPath := path.Join(dir, "ca.pem")
	caKeyPath := path.Join(dir, "ca-key.pem")
	caConfig := tlsutil.CACertConfig{
		CommonName:   "kube-ca",
		Organization: "kube-aws",
	}
	caKey, caCert, err := initTLSCA(caConfig, caKeyPath, caCertPath)
	if err != nil {
		return nil, err
	}

	apiserverCertPath := path.Join(dir, "apiserver.pem")
	apiserverKeyPath := path.Join(dir, "apiserver-key.pem")
	apiserverConfig := tlsutil.ServerCertConfig{
		CommonName: "kube-apiserver",
		DNSNames: []string{
			"kubernetes",
			"kubernetes.default",
			"kubernetes.default.svc",
			"kubernetes.default.svc.cluster.local",
			cfg.ExternalDNSName,
		},
		IPAddresses: []string{
			"10.0.0.50",
			"10.3.0.1",
		},
	}
	if err := initTLSServer(apiserverConfig, caCert, caKey, apiserverKeyPath, apiserverCertPath); err != nil {
		return nil, err
	}

	workerCertPath := path.Join(dir, "worker.pem")
	workerKeyPath := path.Join(dir, "worker-key.pem")
	workerConfig := tlsutil.ClientCertConfig{
		CommonName: "kube-worker",
		DNSNames: []string{
			"*.*.compute.internal", // *.<region>.compute.internal
			"*.ec2.internal",       // for us-east-1
		},
	}
	if err := initTLSClient(workerConfig, caCert, caKey, workerKeyPath, workerCertPath); err != nil {
		return nil, err
	}

	adminCertPath := path.Join(dir, "admin.pem")
	adminKeyPath := path.Join(dir, "admin-key.pem")
	adminConfig := tlsutil.ClientCertConfig{
		CommonName: "kube-admin",
	}
	if err := initTLSClient(adminConfig, caCert, caKey, adminKeyPath, adminCertPath); err != nil {
		return nil, err
	}

	tlsConfig := cluster.TLSConfig{
		CACertFile:        caCertPath,
		CACert:            mustReadFile(caCertPath),
		APIServerCertFile: apiserverCertPath,
		APIServerCert:     mustReadFile(apiserverCertPath),
		APIServerKeyFile:  apiserverKeyPath,
		APIServerKey:      mustReadFile(apiserverKeyPath),
		WorkerCertFile:    workerCertPath,
		WorkerCert:        mustReadFile(workerCertPath),
		WorkerKeyFile:     workerKeyPath,
		WorkerKey:         mustReadFile(workerKeyPath),
		AdminCertFile:     adminCertPath,
		AdminCert:         mustReadFile(adminCertPath),
		AdminKeyFile:      adminKeyPath,
		AdminKey:          mustReadFile(adminKeyPath),
	}

	return &tlsConfig, nil
}

func initTLSCA(cfg tlsutil.CACertConfig, keyPath, certPath string) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	cert, err := tlsutil.NewSelfSignedCACertificate(cfg, key)
	if err != nil {
		return nil, nil, err
	}

	if err := writeKey(keyPath, key); err != nil {
		return nil, nil, err
	}
	if err := writeCert(certPath, cert); err != nil {
		return nil, nil, err
	}

	return key, cert, nil
}

func initTLSServer(cfg tlsutil.ServerCertConfig, caCert *x509.Certificate, caKey *rsa.PrivateKey, keyPath, certPath string) error {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return err
	}

	cert, err := tlsutil.NewSignedServerCertificate(cfg, key, caCert, caKey)
	if err != nil {
		return err
	}

	if err := writeKey(keyPath, key); err != nil {
		return err
	}
	if err := writeCert(certPath, cert); err != nil {
		return err
	}

	return nil
}

func initTLSClient(cfg tlsutil.ClientCertConfig, caCert *x509.Certificate, caKey *rsa.PrivateKey, keyPath, certPath string) error {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return err
	}

	cert, err := tlsutil.NewSignedClientCertificate(cfg, key, caCert, caKey)
	if err != nil {
		return err
	}

	if err := writeKey(keyPath, key); err != nil {
		return err
	}
	if err := writeCert(certPath, cert); err != nil {
		return err
	}

	return nil
}

func writeCert(certPath string, cert *x509.Certificate) error {
	f, err := os.OpenFile(certPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return tlsutil.WriteCertificatePEMBlock(f, cert)
}

func writeKey(keyPath string, key *rsa.PrivateKey) error {
	f, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY, 0400)
	if err != nil {
		return err
	}
	defer f.Close()

	return tlsutil.WritePrivateKeyPEMBlock(f, key)
}
