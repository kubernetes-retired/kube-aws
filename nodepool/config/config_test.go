package config

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/kms"
	"io/ioutil"
	"os"
	"testing"
)

type dummyEncryptService struct{}

func (d dummyEncryptService) Encrypt(input *kms.EncryptInput) (*kms.EncryptOutput, error) {
	output := kms.EncryptOutput{
		CiphertextBlob: input.Plaintext,
	}
	return &output, nil
}

const insufficientConfigYaml = `clusterName: mycluster
nodePoolName: myculster-pool1
externalDNSName: test.staging.core-os.net
keyName: test-key-name
kmsKeyArn: "arn:aws:kms:us-west-1:xxxxxxxxx:key/xxxxxxxxxxxxxxxxxxx"
region: us-west-1
`

const availabilityZoneConfig = `
availabilityZone: us-west-1c
`

const singleAzConfigYaml = insufficientConfigYaml + availabilityZoneConfig

func withDummyCredentials(fn func(dir string)) {
	if _, err := ioutil.ReadDir("temp"); err != nil {
		if err := os.Mkdir("temp", 0755); err != nil {
			panic(err)
		}
	}

	dir, err := ioutil.TempDir("temp", "dummy-credentials")

	if err != nil {
		panic(err)
	}

	defer os.Remove(dir)

	for _, pairName := range []string{"ca", "apiserver", "worker", "admin", "etcd", "etcd-client"} {
		certFile := fmt.Sprintf("%s/%s.pem", dir, pairName)

		if err := ioutil.WriteFile(certFile, []byte("dummycert"), 0644); err != nil {
			panic(err)
		}

		defer os.Remove(certFile)

		keyFile := fmt.Sprintf("%s/%s-key.pem", dir, pairName)

		if err := ioutil.WriteFile(keyFile, []byte("dummykey"), 0644); err != nil {
			panic(err)
		}

		defer os.Remove(keyFile)
	}

	fn(dir)
}

func TestConfig(t *testing.T) {
	minimalValidConfigYaml := insufficientConfigYaml + `
availabilityZone: us-west-1c
dnsServiceIP: "10.3.0.10"
etcdEndpoints: "10.0.0.1"
`
	validCases := []struct {
		context    string
		configYaml string
	}{
		{
			context:    "WithMinimalValidConfig",
			configYaml: minimalValidConfigYaml,
		},
		{
			context: "WithVpcIdSpecified",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
`,
		},
		{
			context: "WithVpcIdAndRouteTableIdSpecified",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
routeTableId: rtb-1a2b3c4d
`,
		},
	}

	for _, validCase := range validCases {
		t.Run(validCase.context, func(t *testing.T) {
			configBytes := validCase.configYaml
			providedConfig, err := ClusterFromBytes([]byte(configBytes))
			if err != nil {
				t.Errorf("failed to parse config %s: %v", configBytes, err)
				t.FailNow()
			}
			providedConfig.providedEncryptService = dummyEncryptService{}

			withDummyCredentials(func(dummyTlsAssetsDir string) {
				var stackTemplateOptions = StackTemplateOptions{
					TLSAssetsDir:          dummyTlsAssetsDir,
					WorkerTmplFile:        "../../config/templates/cloud-config-worker",
					StackTemplateTmplFile: "templates/stack-template.json",
				}

				t.Run("ValidateUserData", func(t *testing.T) {
					if err := providedConfig.ValidateUserData(stackTemplateOptions); err != nil {
						t.Errorf("failed to validate user data: %v", err)
					}
				})

				t.Run("RenderStackTemplate", func(t *testing.T) {
					if _, err := providedConfig.RenderStackTemplate(stackTemplateOptions); err != nil {
						t.Errorf("failed to render stack template: %v", err)
					}
				})
			})
		})
	}

	parseErrorCases := []struct {
		context    string
		configYaml string
	}{
		{
			context: "WithVpcIdAndVPCCIDRSpecified",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
# vpcCIDR (10.1.0.0/16) does not contain instanceCIDR (10.0.1.0/24)
vpcCIDR: "10.1.0.0/16"
`,
		},
	}

	for _, invalidCase := range parseErrorCases {
		t.Run(invalidCase.context, func(t *testing.T) {
			configBytes := invalidCase.configYaml
			providedConfig, err := ClusterFromBytes([]byte(configBytes))
			if err == nil {
				t.Errorf("expected to fail parsing config %s: %v", configBytes, providedConfig)
				t.FailNow()
			}
		})
	}
}
