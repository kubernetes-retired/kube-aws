package config

import (
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/coreos/kube-aws/test/helper"
	"reflect"
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

type ConfigTester func(c *ProvidedConfig, t *testing.T)

func TestConfig(t *testing.T) {
	minimalValidConfigYaml := insufficientConfigYaml + `
availabilityZone: us-west-1c
dnsServiceIP: "10.3.0.10"
etcdEndpoints: "10.0.0.1"
`
	hasDefaultLaunchSpecifications := func(c *ProvidedConfig, t *testing.T) {
		expected := []LaunchSpecification{
			{
				WeightedCapacity: 1,
				InstanceType:     "m3.medium",
				RootVolumeSize:   30,
				RootVolumeIOPS:   0,
				RootVolumeType:   "gp2",
			},
			{
				WeightedCapacity: 2,
				InstanceType:     "m3.large",
				RootVolumeSize:   60,
				RootVolumeIOPS:   0,
				RootVolumeType:   "gp2",
			},
			{
				WeightedCapacity: 2,
				InstanceType:     "m4.large",
				RootVolumeSize:   60,
				RootVolumeIOPS:   0,
				RootVolumeType:   "gp2",
			},
		}
		actual := c.Worker.SpotFleet.LaunchSpecifications
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf(
				"LaunchSpecifications didn't match: expected=%v actual=%v",
				expected,
				actual,
			)
		}
	}

	validCases := []struct {
		context              string
		configYaml           string
		assertProvidedConfig []ConfigTester
	}{
		{
			context:    "WithMinimalValidConfig",
			configYaml: minimalValidConfigYaml,
			assertProvidedConfig: []ConfigTester{
				hasDefaultLaunchSpecifications,
			}},
		{
			context: "WithVpcIdSpecified",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
`,
			assertProvidedConfig: []ConfigTester{
				hasDefaultLaunchSpecifications,
			},
		},
		{
			context: "WithVpcIdAndRouteTableIdSpecified",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
routeTableId: rtb-1a2b3c4d
`,
			assertProvidedConfig: []ConfigTester{
				hasDefaultLaunchSpecifications,
			},
		},
		{
			context: "WithSpotFleetEnabled",
			configYaml: minimalValidConfigYaml + `
worker:
  spotFleet:
    targetCapacity: 10
`,
			assertProvidedConfig: []ConfigTester{
				hasDefaultLaunchSpecifications,
			},
		},
		{
			context: "WithSpotFleetWithCustomGp2RootVolumeSettings",
			configYaml: minimalValidConfigYaml + `
worker:
  spotFleet:
    targetCapacity: 10
    unitRootVolumeSize: 40
    launchSpecifications:
    - weightedCapacity: 1
      instanceType: m3.medium
    - weightedCapacity: 2
      instanceType: m3.large
      rootVolumeSize: 100
`,
			assertProvidedConfig: []ConfigTester{
				func(c *ProvidedConfig, t *testing.T) {
					expected := []LaunchSpecification{
						{
							WeightedCapacity: 1,
							InstanceType:     "m3.medium",
							// RootVolumeSize was not specified in the configYaml but should default to workerRootVolumeSize * weightedCapacity hence:
							RootVolumeSize: 40,
							RootVolumeIOPS: 0,
							// RootVolumeType was not specified in the configYaml but should default to:
							RootVolumeType: "gp2",
						},
						{
							WeightedCapacity: 2,
							InstanceType:     "m3.large",
							RootVolumeSize:   100,
							RootVolumeIOPS:   0,
							RootVolumeType:   "gp2",
						},
					}
					actual := c.Worker.SpotFleet.LaunchSpecifications
					if !reflect.DeepEqual(expected, actual) {
						t.Errorf(
							"LaunchSpecifications didn't match: expected=%v actual=%v",
							expected,
							actual,
						)
					}
				},
			},
		},
		{
			context: "WithSpotFleetWithCustomIo1RootVolumeSettings",
			configYaml: minimalValidConfigYaml + `
worker:
  spotFleet:
    targetCapacity: 10
    rootVolumeType: io1
    unitRootVolumeSize: 40
    unitRootVolumeIOPS: 100
    launchSpecifications:
    - weightedCapacity: 1
      instanceType: m3.medium
    - weightedCapacity: 2
      instanceType: m3.large
      rootVolumeIOPS: 500
`,
			assertProvidedConfig: []ConfigTester{
				func(c *ProvidedConfig, t *testing.T) {
					expected := []LaunchSpecification{
						{
							WeightedCapacity: 1,
							InstanceType:     "m3.medium",
							// RootVolumeSize was not specified in the configYaml but should default to workerRootVolumeSize * weightedCapacity hence:
							RootVolumeSize: 40,
							// RootVolumeIOPS was not specified in the configYaml but should default to workerRootVolumeIOPS * weightedCapacity hence:
							RootVolumeIOPS: 100,
							// RootVolumeType was not specified in the configYaml but should default to:
							RootVolumeType: "io1",
						},
						{
							WeightedCapacity: 2,
							InstanceType:     "m3.large",
							RootVolumeSize:   80,
							RootVolumeIOPS:   500,
							// RootVolumeType was not specified in the configYaml but should default to:
							RootVolumeType: "io1",
						},
					}
					actual := c.Worker.SpotFleet.LaunchSpecifications
					if !reflect.DeepEqual(expected, actual) {
						t.Errorf(
							"LaunchSpecifications didn't match: expected=%v actual=%v",
							expected,
							actual,
						)
					}
				},
			},
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

			t.Run("AssertProvidedConfig", func(t *testing.T) {
				for _, assertion := range validCase.assertProvidedConfig {
					assertion(providedConfig, t)
				}
			})

			helper.WithDummyCredentials(func(dummyTlsAssetsDir string) {
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
		{
			context: "WithSpotFleetWithInvalidRootVolumeType",
			configYaml: minimalValidConfigYaml + `
worker:
  spotFleet:
    targetCapacity: 10
    launchSpecifications:
    - weightedCapacity: 1
      instanceType: m3.medium
      rootVolumeType: foo
`,
		},
		{
			context: "WithSpotFleetWithInvalidRootVolumeIOPS",
			configYaml: minimalValidConfigYaml + `
worker:
  spotFleet:
    targetCapacity: 10
    launchSpecifications:
    - weightedCapacity: 1
      instanceType: m3.medium
      rootVolumeType: io1
      # must be 100~2000
      rootVolumeIOPS: 50
`,
		},
		{
			context: "WithSpotFleetWithInvalidRootVolumeTypeAndIOPSCombination",
			configYaml: minimalValidConfigYaml + `
worker:
  spotFleet:
    targetCapacity: 10
    launchSpecifications:
    - weightedCapacity: 1
      instanceType: m3.medium
      rootVolumeType: gp2
      rootVolumeIOPS: 1000
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
