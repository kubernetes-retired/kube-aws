package cluster

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/config"
)

type dummyEC2Service map[string]struct {
	cidr        string
	subnetCidrs []string
}

func (svc dummyEC2Service) DescribeVpcs(input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
	output := ec2.DescribeVpcsOutput{}
	for _, vpcId := range input.VpcIds {
		if vpc, ok := svc[*vpcId]; ok {
			output.Vpcs = append(output.Vpcs, &ec2.Vpc{
				VpcId:     vpcId,
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

	for _, vpcId := range vpcIds {
		if vpc, ok := svc[vpcId]; ok {
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

func TestExistingVPCValidation(t *testing.T) {
	minimalConfigYaml := `
externalDNSName: test-external-dns-name
keyName: test-key-name
region: us-west-1
availabilityZone: us-west-1c
clusterName: test-cluster-name
kmsKeyArn: "arn:aws:kms:us-west-1:xxxxxxxxx:key/xxxxxxxxxxxxxxxxxxx"
`

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
