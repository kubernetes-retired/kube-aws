package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"crypto/rsa"
	"crypto/x509"

	"path"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/config"
	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/tlsutil"
	"github.com/spf13/cobra"
)

var (
	cmdRender = &cobra.Command{
		Use:          "render",
		Short:        "Render a CloudFormation template",
		Long:         ``,
		RunE:         runCmdRender,
		SilenceUsage: true,
	}
	renderOpts = struct {
		generateCredentials bool
		generateCA          bool
		caKeyPath           string
		caCertPath          string
	}{}
)

func init() {
	cmdRoot.AddCommand(cmdRender)
	cmdRender.Flags().BoolVar(&renderOpts.generateCredentials, "generate-credentials", false, "generate new cluster TLS assets")
	cmdRender.Flags().BoolVar(&renderOpts.generateCA, "generate-ca", false, "if generating credentials, generate root CA key and cert. NOT RECOMMENDED FOR PRODUCTION USE- use '-ca-key-path' and '-ca-cert-path' options to provide your own certificate authority assets")
	cmdRender.Flags().StringVar(&renderOpts.caKeyPath, "ca-key-path", "./credentials/ca-key.pem", "path to pem-encoded CA RSA key")
	cmdRender.Flags().StringVar(&renderOpts.caCertPath, "ca-cert-path", "./credentials/ca.pem", "path to pem-encoded CA x509 certificate")
}

func runCmdRender(cmd *cobra.Command, args []string) error {
	// Read the config from file.
	cluster, err := config.ClusterFromFile(configPath)
	if err != nil {
		return fmt.Errorf("Failed to read cluster config: %v", err)
	}

	if renderOpts.generateCredentials {
		fmt.Printf("Generating TLS credentials...\n")
		var caKey *rsa.PrivateKey
		var caCert *x509.Certificate
		if renderOpts.generateCA {
			var err error
			caKey, caCert, err = config.NewTLSCA()
			if err != nil {
				return fmt.Errorf("failed generating cluster CA: %v", err)
			}
			fmt.Printf("-> Generating new TLS CA\n")
		} else {
			fmt.Printf("-> Parsing existing TLS CA\n")
			if caKeyBytes, err := ioutil.ReadFile(renderOpts.caKeyPath); err != nil {
				return fmt.Errorf("failed reading ca key file %s : %v", renderOpts.caKeyPath, err)
			} else {
				if caKey, err = tlsutil.DecodePrivateKeyPEM(caKeyBytes); err != nil {
					return fmt.Errorf("failed parsing ca key: %v", err)
				}
			}
			if caCertBytes, err := ioutil.ReadFile(renderOpts.caCertPath); err != nil {
				return fmt.Errorf("failed reading ca cert file %s : %v", renderOpts.caCertPath, err)
			} else {
				if caCert, err = tlsutil.DecodeCertificatePEM(caCertBytes); err != nil {
					return fmt.Errorf("failed parsing ca cert: %v", err)
				}
			}
		}
		fmt.Printf("-> Generating new TLS assets\n")
		assets, err := cluster.NewTLSAssets(caKey, caCert)
		if err != nil {
			return fmt.Errorf("Error generating default assets: %v", err)
		}
		if err := os.MkdirAll("credentials", 0700); err != nil {
			return err
		}
		if err := assets.WriteToDir("./credentials", renderOpts.generateCA); err != nil {
			return fmt.Errorf("Error create assets: %v", err)
		}
	}
	fmt.Printf("WARNING: The generated client TLS CA cert expires in %v days and the server and client cert expire in %v days. It is recommended that you create your own TLS infrastructure for revocation and rotation of keys before using in prod\n", cluster.TLSCADurationDays, cluster.TLSCertDurationDays)

	// Create a Config and attempt to render a kubeconfig for it.
	cfg, err := cluster.Config()
	if err != nil {
		return fmt.Errorf("Failed to create config: %v", err)
	}
	tmpl, err := template.New("kubeconfig.yaml").Parse(string(config.KubeConfigTemplate))
	if err != nil {
		return fmt.Errorf("Failed to parse default kubeconfig template: %v", err)
	}
	var kubeconfig bytes.Buffer
	if err := tmpl.Execute(&kubeconfig, cfg); err != nil {
		return fmt.Errorf("Failed to render kubeconfig: %v", err)
	}

	// Write all assets to disk.
	files := []struct {
		name string
		data []byte
		mode os.FileMode
	}{
		{"credentials/.gitignore", []byte("*"), 0644},
		{"userdata/cloud-config-controller", config.CloudConfigController, 0644},
		{"userdata/cloud-config-worker", config.CloudConfigWorker, 0644},
		{"userdata/cloud-config-etcd", config.CloudConfigEtcd, 0644},
		{"stack-template.json", config.StackTemplateTemplate, 0644},
		{"kubeconfig", kubeconfig.Bytes(), 0600},
	}
	for _, file := range files {
		if err := os.MkdirAll(path.Dir(file.name), 0755); err != nil {
			return err
		}

		if err := ioutil.WriteFile(file.name, file.data, file.mode); err != nil {
			return err
		}
	}

	successMsg :=
		`Success! Stack rendered to stack-template.json.

Next steps:
1. (Optional) Validate your changes to %s with "kube-aws validate"
2. (Optional) Further customize the cluster by modifying stack-template.json or files in ./userdata.
3. Start the cluster with "kube-aws up".
`

	fmt.Printf(successMsg, configPath)
	return nil
}
