package cluster

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/config"
)

const minimalConfigYaml = `
externalDNSName: test.staging.core-os.net
keyName: test-key-name
region: us-west-1
availabilityZone: us-west-1c
clusterName: test-cluster-name
kmsKeyArn: "arn:aws:kms:us-west-1:xxxxxxxxx:key/xxxxxxxxxxxxxxxxxxx"
`

type VPC struct {
	cidr        string
	subnetCidrs []string
}

type dummyEC2Service struct {
	VPCs     map[string]VPC
	KeyPairs map[string]bool
}

func (svc dummyEC2Service) DescribeVpcs(input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
	output := ec2.DescribeVpcsOutput{}
	for _, vpcID := range input.VpcIds {
		if vpc, ok := svc.VPCs[*vpcID]; ok {
			output.Vpcs = append(output.Vpcs, &ec2.Vpc{
				VpcId:     vpcID,
				CidrBlock: aws.String(vpc.cidr),
			})
		}
	}

	return &output, nil
}

func (svc dummyEC2Service) DescribeSubnets(input *ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error) {
	output := ec2.DescribeSubnetsOutput{}

	var vpcIds []string
	for _, filter := range input.Filters {
		if *filter.Name == "vpc-id" {
			for _, value := range filter.Values {
				vpcIds = append(vpcIds, *value)
			}
		}
	}

	for _, vpcID := range vpcIds {
		if vpc, ok := svc.VPCs[vpcID]; ok {
			for _, subnetCidr := range vpc.subnetCidrs {
				output.Subnets = append(
					output.Subnets,
					&ec2.Subnet{CidrBlock: aws.String(subnetCidr)},
				)
			}
		}
	}

	return &output, nil
}

func (svc dummyEC2Service) DescribeKeyPairs(input *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
	output := &ec2.DescribeKeyPairsOutput{}

	for _, keyName := range input.KeyNames {
		if _, ok := svc.KeyPairs[*keyName]; ok {
			output.KeyPairs = append(output.KeyPairs, &ec2.KeyPairInfo{
				KeyName: keyName,
			})
		} else {
			return nil, awserr.New("InvalidKeyPair.NotFound", "", errors.New(""))
		}
	}

	return output, nil
}

func TestExistingVPCValidation(t *testing.T) {

	goodExistingVPCConfigs := []string{
		``, //Tests default create VPC mode, which bypasses existing VPC validation
		`
vpcCIDR: 10.5.0.0/16
vpcId: vpc-xxx1
routeTableId: rtb-xxxxxx
instanceCIDR: 10.5.11.0/24
controllerIP: 10.5.11.10
`, `
vpcCIDR: 192.168.1.0/24
vpcId: vpc-xxx2
instanceCIDR: 192.168.1.50/28
controllerIP: 192.168.1.50
`,
	}

	badExistingVPCConfigs := []string{
		`
vpcCIDR: 10.0.0.0/16
vpcId: vpc-xxx3 #vpc does not exist
instanceCIDR: 10.0.0.0/24
controllerIP: 10.0.0.50
routeTableId: rtb-xxxxxx
`, `
vpcCIDR: 10.10.0.0/16 #vpc cidr does match existing vpc-xxx1
vpcId: vpc-xxx1
instanceCIDR: 10.10.0.0/24
controllerIP: 10.10.0.50
routeTableId: rtb-xxxxxx
`, `
vpcCIDR: 10.5.0.0/16
instanceCIDR: 10.5.2.0/28 #instance cidr conflicts with existing subnet
controllerIP: 10.5.2.10
vpcId: vpc-xxx1
routeTableId: rtb-xxxxxx
`, `
vpcCIDR: 192.168.1.0/24
instanceCIDR: 192.168.1.100/26 #instance cidr conflicts with existing subnet
controllerIP: 192.168.1.80
vpcId: vpc-xxx2
routeTableId: rtb-xxxxxx
`,
	}

	ec2Service := dummyEC2Service{
		VPCs: map[string]VPC{
			"vpc-xxx1": {
				cidr: "10.5.0.0/16",
				subnetCidrs: []string{
					"10.5.1.0/24",
					"10.5.2.0/24",
					"10.5.10.100/29",
				},
			},
			"vpc-xxx2": {
				cidr: "192.168.1.0/24",
				subnetCidrs: []string{
					"192.168.1.100/28",
					"192.168.1.150/28",
					"192.168.1.200/28",
				},
			},
		},
	}

	validateCluster := func(networkConfig string) error {
		configBody := minimalConfigYaml + networkConfig
		clusterConfig, err := config.ClusterFromBytes([]byte(configBody))
		if err != nil {
			t.Errorf("could not get valid cluster config: %v", err)
			return nil
		}

		cluster := &Cluster{
			Cluster: *clusterConfig,
		}

		return cluster.validateExistingVPCState(ec2Service)
	}

	for _, networkConfig := range goodExistingVPCConfigs {
		if err := validateCluster(networkConfig); err != nil {
			t.Errorf("Correct config tested invalid: %s\n%s", err, networkConfig)
		}
	}

	for _, networkConfig := range badExistingVPCConfigs {
		if err := validateCluster(networkConfig); err == nil {
			t.Errorf("Incorrect config tested valid, expected error:\n%s", networkConfig)
		}
	}
}

func TestValidateKeyPair(t *testing.T) {

	clusterConfig, err := config.ClusterFromBytes([]byte(minimalConfigYaml))
	if err != nil {
		t.Errorf("could not get valid cluster config: %v", err)
	}

	c := &Cluster{Cluster: *clusterConfig}

	ec2Svc := dummyEC2Service{}
	ec2Svc.KeyPairs = map[string]bool{
		c.KeyName: true,
	}

	if err := c.validateKeyPair(ec2Svc); err != nil {
		t.Errorf("returned an error for valid key")
	}

	c.KeyName = "invalidKeyName"
	if err := c.validateKeyPair(ec2Svc); err == nil {
		t.Errorf("failed to catch invalid key \"%s\"", c.KeyName)
	}
}

type Zone struct {
	Id  string
	DNS string
}

type dummyR53Service struct {
	HostedZones        []Zone
	ResourceRecordSets map[string]string
}

func (r53 dummyR53Service) ListHostedZonesByName(input *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
	output := &route53.ListHostedZonesByNameOutput{}
	for _, zone := range r53.HostedZones {
		if zone.DNS == *input.DNSName {
			output.HostedZones = append(output.HostedZones, &route53.HostedZone{
				Name: aws.String(zone.DNS),
				Id:   aws.String(zone.Id),
			})
		}
	}
	return output, nil
}

func (r53 dummyR53Service) ListResourceRecordSets(input *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error) {
	output := &route53.ListResourceRecordSetsOutput{}
	if name, ok := r53.ResourceRecordSets[*input.HostedZoneId]; ok {
		output.ResourceRecordSets = []*route53.ResourceRecordSet{
			&route53.ResourceRecordSet{
				Name: aws.String(name),
			},
		}
	}
	return output, nil
}

func TestValidateDNSConfig(t *testing.T) {
	dnsConfig := `
createRecordSet: true
recordSetTTL: 60
hostedZone: staging.core-os.net
`

	configBody := minimalConfigYaml + dnsConfig
	clusterConfig, err := config.ClusterFromBytes([]byte(configBody))
	if err != nil {
		t.Errorf("could not get valid cluster config: %v", err)
	}
	c := &Cluster{Cluster: *clusterConfig}

	r53 := dummyR53Service{
		HostedZones: []Zone{
			Zone{
				Id:  "staging_id",
				DNS: "staging.core-os.net.",
			},
		},
		ResourceRecordSets: map[string]string{
			"staging_id": "existing-record.staging.core-os.net.",
		},
	}

	if err := c.validateDNSConfig(r53); err != nil {
		t.Errorf("returned error for valid config: %v", err)
	}

	c.HostedZone = "non-existant-zone"
	if err := c.validateDNSConfig(r53); err == nil {
		t.Errorf("failed to catch non-existent hosted zone")
	}

	c.HostedZone = "staging.core-os.net"
	c.ExternalDNSName = "existing-record.staging.core-os.net"
	if err := c.validateDNSConfig(r53); err == nil {
		t.Errorf("failed to catch already existing ExternalDNSName")
	}
}
