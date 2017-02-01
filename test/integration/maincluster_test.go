package integration

import (
	"fmt"
	"github.com/coreos/kube-aws/cluster"
	"github.com/coreos/kube-aws/config"
	"github.com/coreos/kube-aws/model"
	"github.com/coreos/kube-aws/test/helper"
	"os"
	"reflect"
	"strings"
	"testing"
)

type ConfigTester func(c *config.Cluster, t *testing.T)

// Integration testing with real AWS services including S3, KMS, CloudFormation
func TestMainClusterConfig(t *testing.T) {
	hasDefaultEtcdSettings := func(c *config.Cluster, t *testing.T) {
		subnet1 := model.NewPublicSubnet("us-west-1c", "10.0.0.0/24")
		subnet1.Name = "Subnet0"
		expected := config.EtcdSettings{
			Etcd: model.Etcd{
				Subnets: []model.Subnet{
					subnet1,
				},
			},
			EtcdCount:               1,
			EtcdInstanceType:        "t2.medium",
			EtcdRootVolumeSize:      30,
			EtcdRootVolumeType:      "gp2",
			EtcdRootVolumeIOPS:      0,
			EtcdDataVolumeSize:      30,
			EtcdDataVolumeType:      "gp2",
			EtcdDataVolumeIOPS:      0,
			EtcdDataVolumeEphemeral: false,
			EtcdTenancy:             "default",
		}
		actual := c.EtcdSettings
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf(
				"EtcdSettings didn't match: expected=%v actual=%v",
				expected,
				actual,
			)
		}
	}

	hasDefaultExperimentalFeatures := func(c *config.Cluster, t *testing.T) {
		expected := config.Experimental{
			AuditLog: config.AuditLog{
				Enabled: false,
				MaxAge:  30,
				LogPath: "/dev/stdout",
			},
			AwsEnvironment: config.AwsEnvironment{
				Enabled: false,
			},
			AwsNodeLabels: config.AwsNodeLabels{
				Enabled: false,
			},
			EphemeralImageStorage: config.EphemeralImageStorage{
				Enabled:    false,
				Disk:       "xvdb",
				Filesystem: "xfs",
			},
			LoadBalancer: config.LoadBalancer{
				Enabled: false,
			},
			NodeDrainer: config.NodeDrainer{
				Enabled: false,
			},
			NodeLabels: config.NodeLabels{},
			Taints:     []config.Taint{},
			WaitSignal: config.WaitSignal{
				Enabled:      false,
				MaxBatchSize: 1,
			},
		}

		actual := c.Experimental

		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("experimental settings didn't match :\nexpected=%v\nactual=%v", expected, actual)
		}
	}

	everyPublicSubnetHasRouteToIGW := func(c *config.Cluster, t *testing.T) {
		for i, s := range c.PublicSubnets() {
			if !s.ManageRouteToInternet() {
				t.Errorf("Public subnet %d should have a route to the IGW but it doesn't: %+v", i, s)
			}
		}
	}

	kubeAwsSettings := newKubeAwsSettingsFromEnv(t)

	minimalValidConfigYaml := kubeAwsSettings.mainClusterYaml + `
availabilityZone: us-west-1c
`
	validCases := []struct {
		context      string
		configYaml   string
		assertConfig []ConfigTester
	}{
		{
			context: "WithExperimentalFeatures",
			configYaml: minimalValidConfigYaml + `
experimental:
  auditLog:
    enabled: true
    maxage: 100
    logpath: "/var/log/audit.log"
  awsEnvironment:
    enabled: true
    environment:
      CFNSTACK: '{ "Ref" : "AWS::StackId" }'
  awsNodeLabels:
    enabled: true
  ephemeralImageStorage:
    enabled: true
  loadBalancer:
    enabled: true
    names:
      - manuallymanagedlb
    securityGroupIds:
      - sg-12345678
  nodeDrainer:
    enabled: true
  nodeLabels:
    kube-aws.coreos.com/role: worker
  plugins:
    rbac:
      enabled: true
  taints:
    - key: reservation
      value: spot
      effect: NoSchedule
  waitSignal:
    enabled: true
`,
			assertConfig: []ConfigTester{
				hasDefaultEtcdSettings,
				func(c *config.Cluster, t *testing.T) {
					expected := config.Experimental{
						AuditLog: config.AuditLog{
							Enabled: true,
							MaxAge:  100,
							LogPath: "/var/log/audit.log",
						},
						AwsEnvironment: config.AwsEnvironment{
							Enabled: true,
							Environment: map[string]string{
								"CFNSTACK": `{ "Ref" : "AWS::StackId" }`,
							},
						},
						AwsNodeLabels: config.AwsNodeLabels{
							Enabled: true,
						},
						EphemeralImageStorage: config.EphemeralImageStorage{
							Enabled:    true,
							Disk:       "xvdb",
							Filesystem: "xfs",
						},
						LoadBalancer: config.LoadBalancer{
							Enabled:          true,
							Names:            []string{"manuallymanagedlb"},
							SecurityGroupIds: []string{"sg-12345678"},
						},
						NodeDrainer: config.NodeDrainer{
							Enabled: true,
						},
						NodeLabels: config.NodeLabels{
							"kube-aws.coreos.com/role": "worker",
						},
						Plugins: config.Plugins{
							Rbac: config.Rbac{
								Enabled: true,
							},
						},
						Taints: []config.Taint{
							{Key: "reservation", Value: "spot", Effect: "NoSchedule"},
						},
						WaitSignal: config.WaitSignal{
							Enabled:      true,
							MaxBatchSize: 1,
						},
					}

					actual := c.Experimental

					if !reflect.DeepEqual(expected, actual) {
						t.Errorf("experimental settings didn't match : expected=%v actual=%v", expected, actual)
					}
				},
			},
		},
		{
			context:    "WithMinimalValidConfig",
			configYaml: minimalValidConfigYaml,
			assertConfig: []ConfigTester{
				hasDefaultEtcdSettings,
				hasDefaultExperimentalFeatures,
			},
		},
		{
			context: "WithNetworkTopologyAllPreconfiguredPrivateDeprecated",
			configYaml: kubeAwsSettings.mainClusterYaml + `
vpcId: vpc-1a2b3c4d
# This, in combination with mapPublicIPs=false, implies that the route table contains a route to a preconfigured NAT gateway
# See https://github.com/coreos/kube-aws/pull/284#issuecomment-276008202
routeTableId: rtb-1a2b3c4d
# This means that all the subnets created by kube-aws should be private
mapPublicIPs: false
# This can't be false because kube-aws won't create public subbnets which are required by an external lb when mapPublicIPs=false
controllerLoadBalancerPrivate: true
subnets:
- availabilityZone: us-west-1a
  instanceCIDR: "10.0.1.0/24"
  # implies
  # private: true
  # routeTable
  #   id: rtb-1a2b3c4d
- availabilityZone: us-west-1b
  instanceCIDR: "10.0.2.0/24"
  # implies
  # private: true
  # routeTable
  #   id: rtb-1a2b3c4d
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				func(c *config.Cluster, t *testing.T) {
					private1 := model.NewPrivateSubnetWithPreconfiguredNATGateway("us-west-1a", "10.0.1.0/24", "rtb-1a2b3c4d")
					private1.Name = "Subnet0"

					private2 := model.NewPrivateSubnetWithPreconfiguredNATGateway("us-west-1b", "10.0.2.0/24", "rtb-1a2b3c4d")
					private2.Name = "Subnet1"

					subnets := []model.Subnet{
						private1,
						private2,
					}
					if !reflect.DeepEqual(c.AllSubnets(), subnets) {
						t.Errorf("Managed subnets didn't match: expected=%+v actual=%+v", subnets, c.AllSubnets())
					}

					privateSubnets := []model.Subnet{
						private1,
						private2,
					}
					if !reflect.DeepEqual(c.Worker.Subnets, privateSubnets) {
						t.Errorf("Worker subnets didn't match: expected=%+v actual=%+v", subnets, c.Worker.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.Subnets, privateSubnets) {
						t.Errorf("Controller subnets didn't match: expected=%+v actual=%+v", privateSubnets, c.Controller.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.LoadBalancer.Subnets, privateSubnets) {
						t.Errorf("Controller loadbalancer subnets didn't match: expected=%+v actual=%+v", privateSubnets, c.Controller.LoadBalancer.Subnets)
					}
					if !reflect.DeepEqual(c.Etcd.Subnets, privateSubnets) {
						t.Errorf("Etcd subnets didn't match: expected=%+v actual=%+v", privateSubnets, c.Etcd.Subnets)
					}

					for i, s := range c.PrivateSubnets() {
						if s.ManageNATGateway() {
							t.Errorf("NAT gateway for the private subnet #%d is externally managed and shouldn't created by kube-aws", i)
						}

						if s.ManageRouteToInternet() {
							t.Errorf("Route to IGW shouldn't be created for a private subnet: %+v", s)
						}
					}

					if len(c.PublicSubnets()) != 0 {
						t.Errorf("Number of public subnets should be zero but it wasn't: %d", len(c.PublicSubnets()))
					}
				},
			},
		},
		{
			context: "WithNetworkTopologyAllPreconfiguredPublicDeprecated",
			configYaml: kubeAwsSettings.mainClusterYaml + `
vpcId: vpc-1a2b3c4d
# This, in combination with mapPublicIPs=true, implies that the route table contains a route to a preconfigured internet gateway
# See https://github.com/coreos/kube-aws/pull/284#issuecomment-276008202
routeTableId: rtb-1a2b3c4d
# This means that all the subnets created by kube-aws should be public
mapPublicIPs: true
# This can't be true because kube-aws won't create private subnets which are required by an internal lb when mapPublicIPs=true
controllerLoadBalancerPrivate: false
# internetGatewayId should be omitted as we assume that the route table specified by routeTableId already contain a route to one
#internetGatewayId:
subnets:
- availabilityZone: us-west-1a
  instanceCIDR: "10.0.1.0/24"
  # #implies
  # private: false
  # routeTable
  #   id: rtb-1a2b3c4d
- availabilityZone: us-west-1b
  instanceCIDR: "10.0.2.0/24"
  # #implies
  # private: false
  # routeTable
  #   id: rtb-1a2b3c4d
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				func(c *config.Cluster, t *testing.T) {
					private1 := model.NewPublicSubnetWithPreconfiguredInternetGateway("us-west-1a", "10.0.1.0/24", "rtb-1a2b3c4d")
					private1.Name = "Subnet0"

					private2 := model.NewPublicSubnetWithPreconfiguredInternetGateway("us-west-1b", "10.0.2.0/24", "rtb-1a2b3c4d")
					private2.Name = "Subnet1"

					subnets := []model.Subnet{
						private1,
						private2,
					}
					if !reflect.DeepEqual(c.AllSubnets(), subnets) {
						t.Errorf("Managed subnets didn't match: expected=%+v actual=%+v", subnets, c.AllSubnets())
					}

					publicSubnets := []model.Subnet{
						private1,
						private2,
					}
					if !reflect.DeepEqual(c.Worker.Subnets, publicSubnets) {
						t.Errorf("Worker subnets didn't match: expected=%+v actual=%+v", subnets, c.Worker.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.Subnets, publicSubnets) {
						t.Errorf("Controller subnets didn't match: expected=%+v actual=%+v", publicSubnets, c.Controller.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.LoadBalancer.Subnets, publicSubnets) {
						t.Errorf("Controller loadbalancer subnets didn't match: expected=%+v actual=%+v", publicSubnets, c.Controller.LoadBalancer.Subnets)
					}
					if !reflect.DeepEqual(c.Etcd.Subnets, publicSubnets) {
						t.Errorf("Etcd subnets didn't match: expected=%+v actual=%+v", publicSubnets, c.Etcd.Subnets)
					}

					for i, s := range c.PublicSubnets() {
						if s.RouteTableID() != "rtb-1a2b3c4d" {
							t.Errorf("Subnet %d should be associated to a route table with an IGW preconfigured but it wasn't", i)
						}

						if s.ManageRouteToInternet() {
							t.Errorf("Route to IGW shouldn't be created for a public subnet with a preconfigured IGW: %+v", s)
						}
					}

					if len(c.PrivateSubnets()) != 0 {
						t.Errorf("Number of private subnets should be zero but it wasn't: %d", len(c.PrivateSubnets()))
					}
				},
			},
		},
		{
			context: "WithNetworkTopologyExplicitSubnets",
			configYaml: kubeAwsSettings.mainClusterYaml + `
vpcId: vpc-1a2b3c4d
# routeTableId must be omitted
# See https://github.com/coreos/kube-aws/pull/284#issuecomment-275962332
# routeTableId: rtb-1a2b3c4d
subnets:
- name: private1
  availabilityZone: us-west-1a
  instanceCIDR: "10.0.1.0/24"
  private: true
- name: private2
  availabilityZone: us-west-1b
  instanceCIDR: "10.0.2.0/24"
  private: true
- name: public1
  availabilityZone: us-west-1a
  instanceCIDR: "10.0.3.0/24"
- name: public2
  availabilityZone: us-west-1b
  instanceCIDR: "10.0.4.0/24"
controller:
  subnets:
  - name: private1
  - name: private2
  loadBalancer:
    subnets:
    - name: public1
    - name: public2
    private: false
etcd:
  subnets:
  - name: private1
  - name: private2
worker:
  subnets:
  - name: public1
  - name: public2
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				everyPublicSubnetHasRouteToIGW,
				func(c *config.Cluster, t *testing.T) {
					private1 := model.NewPrivateSubnet("us-west-1a", "10.0.1.0/24")
					private1.Name = "private1"

					private2 := model.NewPrivateSubnet("us-west-1b", "10.0.2.0/24")
					private2.Name = "private2"

					public1 := model.NewPublicSubnet("us-west-1a", "10.0.3.0/24")
					public1.Name = "public1"

					public2 := model.NewPublicSubnet("us-west-1b", "10.0.4.0/24")
					public2.Name = "public2"

					subnets := []model.Subnet{
						private1,
						private2,
						public1,
						public2,
					}
					if !reflect.DeepEqual(c.AllSubnets(), subnets) {
						t.Errorf("Managed subnets didn't match: expected=%v actual=%v", subnets, c.AllSubnets())
					}

					publicSubnets := []model.Subnet{
						public1,
						public2,
					}
					if !reflect.DeepEqual(c.Worker.Subnets, publicSubnets) {
						t.Errorf("Worker subnets didn't match: expected=%v actual=%v", publicSubnets, c.Worker.Subnets)
					}

					privateSubnets := []model.Subnet{
						private1,
						private2,
					}
					if !reflect.DeepEqual(c.Controller.Subnets, privateSubnets) {
						t.Errorf("Controller subnets didn't match: expected=%v actual=%v", privateSubnets, c.Controller.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.LoadBalancer.Subnets, publicSubnets) {
						t.Errorf("Controller loadbalancer subnets didn't match: expected=%v actual=%v", publicSubnets, c.Controller.LoadBalancer.Subnets)
					}
					if !reflect.DeepEqual(c.Etcd.Subnets, privateSubnets) {
						t.Errorf("Etcd subnets didn't match: expected=%v actual=%v", privateSubnets, c.Etcd.Subnets)
					}

					for i, s := range c.PrivateSubnets() {
						if !s.ManageNATGateway() {
							t.Errorf("NAT gateway for the private subnet #%d should be created by kube-aws but it is not going to be", i)
						}

						if s.ManageRouteToInternet() {
							t.Errorf("Route to IGW shouldn't be created for a private subnet: %v", s)
						}
					}
				},
			},
		},
		{
			context: "WithNetworkTopologyImplicitSubnets",
			configYaml: kubeAwsSettings.mainClusterYaml + `
vpcId: vpc-1a2b3c4d
# routeTableId must be omitted
# See https://github.com/coreos/kube-aws/pull/284#issuecomment-275962332
# routeTableId: rtb-1a2b3c4d
subnets:
- name: private1
  availabilityZone: us-west-1a
  instanceCIDR: "10.0.1.0/24"
  private: true
- name: private2
  availabilityZone: us-west-1b
  instanceCIDR: "10.0.2.0/24"
  private: true
- name: public1
  availabilityZone: us-west-1a
  instanceCIDR: "10.0.3.0/24"
- name: public2
  availabilityZone: us-west-1b
  instanceCIDR: "10.0.4.0/24"
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				everyPublicSubnetHasRouteToIGW,
				func(c *config.Cluster, t *testing.T) {
					private1 := model.NewPrivateSubnet("us-west-1a", "10.0.1.0/24")
					private1.Name = "private1"

					private2 := model.NewPrivateSubnet("us-west-1b", "10.0.2.0/24")
					private2.Name = "private2"

					public1 := model.NewPublicSubnet("us-west-1a", "10.0.3.0/24")
					public1.Name = "public1"

					public2 := model.NewPublicSubnet("us-west-1b", "10.0.4.0/24")
					public2.Name = "public2"

					subnets := []model.Subnet{
						private1,
						private2,
						public1,
						public2,
					}
					if !reflect.DeepEqual(c.AllSubnets(), subnets) {
						t.Errorf("Managed subnets didn't match: expected=%v actual=%v", subnets, c.AllSubnets())
					}

					publicSubnets := []model.Subnet{
						public1,
						public2,
					}
					if !reflect.DeepEqual(c.Worker.Subnets, publicSubnets) {
						t.Errorf("Worker subnets didn't match: expected=%v actual=%v", publicSubnets, c.Worker.Subnets)
					}

					if !reflect.DeepEqual(c.Controller.Subnets, publicSubnets) {
						t.Errorf("Controller subnets didn't match: expected=%v actual=%v", publicSubnets, c.Controller.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.LoadBalancer.Subnets, publicSubnets) {
						t.Errorf("Controller loadbalancer subnets didn't match: expected=%v actual=%v", publicSubnets, c.Controller.LoadBalancer.Subnets)
					}
					if !reflect.DeepEqual(c.Etcd.Subnets, publicSubnets) {
						t.Errorf("Etcd subnets didn't match: expected=%v actual=%v", publicSubnets, c.Etcd.Subnets)
					}

					for i, s := range c.PrivateSubnets() {
						if !s.ManageNATGateway() {
							t.Errorf("NAT gateway for the private subnet #%d should be created by kube-aws but it is not going to be", i)
						}

						if s.ManageRouteToInternet() {
							t.Errorf("Route to IGW shouldn't be created for a private subnet: %v", s)
						}
					}
				},
			},
		},
		{
			context: "WithNetworkTopologyControllerPrivateLB",
			configYaml: kubeAwsSettings.mainClusterYaml + `
vpcId: vpc-1a2b3c4d
# routeTableId must be omitted
# See https://github.com/coreos/kube-aws/pull/284#issuecomment-275962332
# routeTableId: rtb-1a2b3c4d
subnets:
- name: private1
  availabilityZone: us-west-1a
  instanceCIDR: "10.0.1.0/24"
  private: true
- name: private2
  availabilityZone: us-west-1b
  instanceCIDR: "10.0.2.0/24"
  private: true
- name: public1
  availabilityZone: us-west-1a
  instanceCIDR: "10.0.3.0/24"
- name: public2
  availabilityZone: us-west-1b
  instanceCIDR: "10.0.4.0/24"
controller:
  subnets:
  - name: private1
  - name: private2
  loadBalancer:
    private: true
etcd:
  subnets:
  - name: private1
  - name: private2
worker:
  subnets:
  - name: public1
  - name: public2
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				everyPublicSubnetHasRouteToIGW,
				func(c *config.Cluster, t *testing.T) {
					private1 := model.NewPrivateSubnet("us-west-1a", "10.0.1.0/24")
					private1.Name = "private1"

					private2 := model.NewPrivateSubnet("us-west-1b", "10.0.2.0/24")
					private2.Name = "private2"

					public1 := model.NewPublicSubnet("us-west-1a", "10.0.3.0/24")
					public1.Name = "public1"

					public2 := model.NewPublicSubnet("us-west-1b", "10.0.4.0/24")
					public2.Name = "public2"

					subnets := []model.Subnet{
						private1,
						private2,
						public1,
						public2,
					}
					if !reflect.DeepEqual(c.AllSubnets(), subnets) {
						t.Errorf("Managed subnets didn't match: expected=%v actual=%v", subnets, c.AllSubnets())
					}

					publicSubnets := []model.Subnet{
						public1,
						public2,
					}
					if !reflect.DeepEqual(c.Worker.Subnets, publicSubnets) {
						t.Errorf("Worker subnets didn't match: expected=%v actual=%v", publicSubnets, c.Worker.Subnets)
					}

					privateSubnets := []model.Subnet{
						private1,
						private2,
					}
					if !reflect.DeepEqual(c.Controller.Subnets, privateSubnets) {
						t.Errorf("Controller subnets didn't match: expected=%v actual=%v", privateSubnets, c.Controller.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.LoadBalancer.Subnets, privateSubnets) {
						t.Errorf("Controller loadbalancer subnets didn't match: expected=%v actual=%v", privateSubnets, c.Controller.LoadBalancer.Subnets)
					}
					if !reflect.DeepEqual(c.Etcd.Subnets, privateSubnets) {
						t.Errorf("Etcd subnets didn't match: expected=%v actual=%v", privateSubnets, c.Etcd.Subnets)
					}

					for i, s := range c.PrivateSubnets() {
						if !s.ManageNATGateway() {
							t.Errorf("NAT gateway for the private subnet #%d should be created by kube-aws but it is not going to be", i)
						}

						if s.ManageRouteToInternet() {
							t.Errorf("Route to IGW shouldn't be created for a private subnet: %v", s)
						}
					}
				},
			},
		},
		{
			context: "WithNetworkTopologyControllerPublicLB",
			configYaml: kubeAwsSettings.mainClusterYaml + `
vpcId: vpc-1a2b3c4d
# routeTableId must be omitted
# See https://github.com/coreos/kube-aws/pull/284#issuecomment-275962332
# routeTableId: rtb-1a2b3c4d
subnets:
- name: private1
  availabilityZone: us-west-1a
  instanceCIDR: "10.0.1.0/24"
  private: true
- name: private2
  availabilityZone: us-west-1b
  instanceCIDR: "10.0.2.0/24"
  private: true
- name: public1
  availabilityZone: us-west-1a
  instanceCIDR: "10.0.3.0/24"
- name: public2
  availabilityZone: us-west-1b
  instanceCIDR: "10.0.4.0/24"
controller:
  loadBalancer:
    private: false
etcd:
  subnets:
  - name: private1
  - name: private2
worker:
  subnets:
  - name: public1
  - name: public2
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				everyPublicSubnetHasRouteToIGW,
				func(c *config.Cluster, t *testing.T) {
					private1 := model.NewPrivateSubnet("us-west-1a", "10.0.1.0/24")
					private1.Name = "private1"

					private2 := model.NewPrivateSubnet("us-west-1b", "10.0.2.0/24")
					private2.Name = "private2"

					public1 := model.NewPublicSubnet("us-west-1a", "10.0.3.0/24")
					public1.Name = "public1"

					public2 := model.NewPublicSubnet("us-west-1b", "10.0.4.0/24")
					public2.Name = "public2"

					subnets := []model.Subnet{
						private1,
						private2,
						public1,
						public2,
					}
					publicSubnets := []model.Subnet{
						public1,
						public2,
					}
					privateSubnets := []model.Subnet{
						private1,
						private2,
					}

					if !reflect.DeepEqual(c.AllSubnets(), subnets) {
						t.Errorf("Managed subnets didn't match: expected=%v actual=%v", subnets, c.AllSubnets())
					}
					if !reflect.DeepEqual(c.Worker.Subnets, publicSubnets) {
						t.Errorf("Worker subnets didn't match: expected=%v actual=%v", publicSubnets, c.Worker.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.Subnets, publicSubnets) {
						t.Errorf("Controller subnets didn't match: expected=%v actual=%v", privateSubnets, c.Controller.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.LoadBalancer.Subnets, publicSubnets) {
						t.Errorf("Controller loadbalancer subnets didn't match: expected=%v actual=%v", privateSubnets, c.Controller.LoadBalancer.Subnets)
					}
					if !reflect.DeepEqual(c.Etcd.Subnets, privateSubnets) {
						t.Errorf("Etcd subnets didn't match: expected=%v actual=%v", privateSubnets, c.Etcd.Subnets)
					}

					for i, s := range c.PrivateSubnets() {
						if !s.ManageNATGateway() {
							t.Errorf("NAT gateway for the private subnet #%d should be created by kube-aws but it is not going to be", i)
						}

						if s.ManageRouteToInternet() {
							t.Errorf("Route to IGW shouldn't be created for a private subnet: %v", s)
						}
					}
				},
			},
		},
		{
			context: "WithNetworkTopologyExistingSubnets",
			configYaml: kubeAwsSettings.mainClusterYaml + `
vpcId: vpc-1a2b3c4d
subnets:
- name: private1
  availabilityZone: us-west-1a
  id: subnet-1
  private: true
- name: private2
  availabilityZone: us-west-1b
  idFromStackOutput: mycluster-private-subnet-1
  private: true
- name: public1
  availabilityZone: us-west-1a
  id: subnet-2
- name: public2
  availabilityZone: us-west-1b
  idFromStackOutput: mycluster-public-subnet-1
controller:
  loadBalancer:
    private: false
etcd:
  subnets:
  - name: private1
  - name: private2
worker:
  subnets:
  - name: public1
  - name: public2
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				func(c *config.Cluster, t *testing.T) {
					private1 := model.NewExistingPrivateSubnet("us-west-1a", "subnet-1")
					private1.Name = "private1"

					private2 := model.NewImportedPrivateSubnet("us-west-1b", "mycluster-private-subnet-1")
					private2.Name = "private2"

					public1 := model.NewExistingPublicSubnet("us-west-1a", "subnet-2")
					public1.Name = "public1"

					public2 := model.NewImportedPublicSubnet("us-west-1b", "mycluster-public-subnet-1")
					public2.Name = "public2"

					subnets := []model.Subnet{
						private1,
						private2,
						public1,
						public2,
					}
					publicSubnets := []model.Subnet{
						public1,
						public2,
					}
					privateSubnets := []model.Subnet{
						private1,
						private2,
					}

					if !reflect.DeepEqual(c.AllSubnets(), subnets) {
						t.Errorf("Managed subnets didn't match: expected=%v actual=%v", subnets, c.AllSubnets())
					}
					if !reflect.DeepEqual(c.Worker.Subnets, publicSubnets) {
						t.Errorf("Worker subnets didn't match: expected=%v actual=%v", publicSubnets, c.Worker.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.Subnets, publicSubnets) {
						t.Errorf("Controller subnets didn't match: expected=%v actual=%v", privateSubnets, c.Controller.Subnets)
					}
					if !reflect.DeepEqual(c.Controller.LoadBalancer.Subnets, publicSubnets) {
						t.Errorf("Controller loadbalancer subnets didn't match: expected=%v actual=%v", privateSubnets, c.Controller.LoadBalancer.Subnets)
					}
					if !reflect.DeepEqual(c.Etcd.Subnets, privateSubnets) {
						t.Errorf("Etcd subnets didn't match: expected=%v actual=%v", privateSubnets, c.Etcd.Subnets)
					}

					for i, s := range c.PrivateSubnets() {
						if s.ManageNATGateway() {
							t.Errorf("NAT gateway for the existing private subnet #%d should not be created by kube-aws", i)
						}

						if s.ManageRouteToInternet() {
							t.Errorf("Route to IGW shouldn't be created for a private subnet: %v", s)
						}
					}
				},
			},
		},
		{
			context: "WithVpcIdSpecified",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
`,
			assertConfig: []ConfigTester{
				hasDefaultEtcdSettings,
				hasDefaultExperimentalFeatures,
			},
		},
		{
			context: "WithVpcIdAndRouteTableIdSpecified",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
routeTableId: rtb-1a2b3c4d
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				func(c *config.Cluster, t *testing.T) {
					subnet1 := model.NewPublicSubnetWithPreconfiguredInternetGateway("us-west-1c", "10.0.0.0/24", "rtb-1a2b3c4d")
					subnet1.Name = "Subnet0"
					subnets := []model.Subnet{
						subnet1,
					}
					expected := config.EtcdSettings{
						Etcd: model.Etcd{
							Subnets: subnets,
						},
						EtcdCount:               1,
						EtcdInstanceType:        "t2.medium",
						EtcdRootVolumeSize:      30,
						EtcdRootVolumeType:      "gp2",
						EtcdDataVolumeSize:      30,
						EtcdDataVolumeType:      "gp2",
						EtcdDataVolumeEphemeral: false,
						EtcdTenancy:             "default",
					}
					actual := c.EtcdSettings
					if !reflect.DeepEqual(expected, actual) {
						t.Errorf(
							"EtcdSettings didn't match: expected=%v actual=%v",
							expected,
							actual,
						)
					}
				},
			},
		},
		{
			context: "WithWorkerSecurityGroupIds",
			configYaml: minimalValidConfigYaml + `
workerSecurityGroupIds:
  - sg-12345678
  - sg-abcdefab
  - sg-23456789
  - sg-bcdefabc
`,
			assertConfig: []ConfigTester{
				hasDefaultEtcdSettings,
				hasDefaultExperimentalFeatures,
				func(c *config.Cluster, t *testing.T) {
					expectedWorkerSecurityGroupIds := []string{
						`sg-12345678`, `sg-abcdefab`, `sg-23456789`, `sg-bcdefabc`,
					}
					if !reflect.DeepEqual(c.WorkerSecurityGroupIds, expectedWorkerSecurityGroupIds) {
						t.Errorf("WorkerSecurityGroupIds didn't match: expected=%v actual=%v", expectedWorkerSecurityGroupIds, c.WorkerSecurityGroupIds)
					}

					expectedWorkerSecurityGroupRefs := []string{
						`"sg-12345678"`, `"sg-abcdefab"`, `"sg-23456789"`, `"sg-bcdefabc"`,
					}
					if !reflect.DeepEqual(c.WorkerSecurityGroupRefs(), expectedWorkerSecurityGroupRefs) {
						t.Errorf("WorkerSecurityGroupRefs didn't match: expected=%v actual=%v", expectedWorkerSecurityGroupRefs, c.WorkerSecurityGroupRefs())
					}
				},
			},
		},
		{
			context: "WithWorkerAndLBSecurityGroupIds",
			configYaml: minimalValidConfigYaml + `
workerSecurityGroupIds:
  - sg-12345678
  - sg-abcdefab
experimental:
  loadBalancer:
    enabled: true
    securityGroupIds:
      - sg-23456789
      - sg-bcdefabc
`,
			assertConfig: []ConfigTester{
				hasDefaultEtcdSettings,
				func(c *config.Cluster, t *testing.T) {
					expectedWorkerSecurityGroupIds := []string{
						`sg-12345678`, `sg-abcdefab`,
					}
					if !reflect.DeepEqual(c.WorkerSecurityGroupIds, expectedWorkerSecurityGroupIds) {
						t.Errorf("WorkerSecurityGroupIds didn't match: expected=%v actual=%v", expectedWorkerSecurityGroupIds, c.WorkerSecurityGroupIds)
					}

					expectedLBSecurityGroupIds := []string{
						`sg-23456789`, `sg-bcdefabc`,
					}
					if !reflect.DeepEqual(c.Experimental.LoadBalancer.SecurityGroupIds, expectedLBSecurityGroupIds) {
						t.Errorf("LBSecurityGroupIds didn't match: expected=%v actual=%v", expectedLBSecurityGroupIds, c.Experimental.LoadBalancer.SecurityGroupIds)
					}

					expectedWorkerSecurityGroupRefs := []string{
						`"sg-23456789"`, `"sg-bcdefabc"`, `"sg-12345678"`, `"sg-abcdefab"`,
					}
					if !reflect.DeepEqual(c.WorkerSecurityGroupRefs(), expectedWorkerSecurityGroupRefs) {
						t.Errorf("WorkerSecurityGroupRefs didn't match: expected=%v actual=%v", expectedWorkerSecurityGroupRefs, c.WorkerSecurityGroupRefs())
					}
				},
			},
		},
		{
			context: "WithDedicatedInstanceTenancy",
			configYaml: minimalValidConfigYaml + `
workerTenancy: dedicated
controllerTenancy: dedicated
etcdTenancy: dedicated
`,
			assertConfig: []ConfigTester{
				func(c *config.Cluster, t *testing.T) {
					if c.EtcdSettings.EtcdTenancy != "dedicated" {
						t.Errorf("EtcdSettings.EtcdTenancy didn't match: expected=dedicated actual=%s", c.EtcdSettings.EtcdTenancy)
					}
					if c.WorkerTenancy != "dedicated" {
						t.Errorf("WorkerTenancy didn't match: expected=dedicated actual=%s", c.WorkerTenancy)
					}
					if c.ControllerTenancy != "dedicated" {
						t.Errorf("ControllerTenancy didn't match: expected=dedicated actual=%s", c.ControllerTenancy)
					}
				},
			},
		},
		{
			context: "WithEtcdNodesWithCustomEBSVolumes",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
routeTableId: rtb-1a2b3c4d
etcdCount: 2
etcdRootVolumeSize: 101
etcdRootVolumeType: io1
etcdRootVolumeIOPS: 102
etcdDataVolumeSize: 103
etcdDataVolumeType: io1
etcdDataVolumeIOPS: 104
`,
			assertConfig: []ConfigTester{
				hasDefaultExperimentalFeatures,
				func(c *config.Cluster, t *testing.T) {
					subnet1 := model.NewPublicSubnetWithPreconfiguredInternetGateway("us-west-1c", "10.0.0.0/24", "rtb-1a2b3c4d")
					subnet1.Name = "Subnet0"
					subnets := []model.Subnet{
						subnet1,
					}
					expected := config.EtcdSettings{
						Etcd: model.Etcd{
							Subnets: subnets,
						},
						EtcdCount:               2,
						EtcdInstanceType:        "t2.medium",
						EtcdRootVolumeSize:      101,
						EtcdRootVolumeType:      "io1",
						EtcdRootVolumeIOPS:      102,
						EtcdDataVolumeSize:      103,
						EtcdDataVolumeType:      "io1",
						EtcdDataVolumeIOPS:      104,
						EtcdDataVolumeEphemeral: false,
						EtcdTenancy:             "default",
					}
					actual := c.EtcdSettings
					if !reflect.DeepEqual(expected, actual) {
						t.Errorf(
							"EtcdSettings didn't match: expected=%v actual=%v",
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
			providedConfig, err := config.ClusterFromBytesWithEncryptService([]byte(configBytes), helper.DummyEncryptService{})
			if err != nil {
				t.Errorf("failed to parse config %s: %v", configBytes, err)
				t.FailNow()
			}

			t.Run("AssertConfig", func(t *testing.T) {
				for _, assertion := range validCase.assertConfig {
					assertion(providedConfig, t)
				}
			})

			helper.WithDummyCredentials(func(dummyTlsAssetsDir string) {
				s3URI, s3URIExists := os.LookupEnv("KUBE_AWS_S3_DIR_URI")

				var stackTemplateOptions = config.StackTemplateOptions{
					TLSAssetsDir:          dummyTlsAssetsDir,
					ControllerTmplFile:    "../../config/templates/cloud-config-controller",
					WorkerTmplFile:        "../../config/templates/cloud-config-worker",
					EtcdTmplFile:          "../../config/templates/cloud-config-etcd",
					StackTemplateTmplFile: "../../config/templates/stack-template.json",
					S3URI: s3URI,
				}

				cluster, err := cluster.NewCluster(providedConfig, stackTemplateOptions, false)
				if err != nil {
					t.Errorf("failed to create cluster driver : %v", err)
					t.FailNow()
				}

				t.Run("ValidateUserData", func(t *testing.T) {
					if err := cluster.ValidateUserData(); err != nil {
						t.Errorf("failed to validate user data: %v", err)
					}
				})

				t.Run("RenderStackTemplate", func(t *testing.T) {
					if _, err := cluster.RenderStackTemplateAsString(); err != nil {
						t.Errorf("failed to render stack template: %v", err)
					}
				})

				if os.Getenv("KUBE_AWS_INTEGRATION_TEST") == "" {
					t.Skipf("`export KUBE_AWS_INTEGRATION_TEST=1` is required to run integration tests. Skipping.")
					t.SkipNow()
				} else {
					t.Run("ValidateStack", func(t *testing.T) {
						if !s3URIExists {
							t.Errorf("failed to obtain value for KUBE_AWS_S3_DIR_URI")
							t.FailNow()
						}

						report, err := cluster.ValidateStack()

						if err != nil {
							t.Errorf("failed to validate stack: %s %v", report, err)
						}
					})
				}
			})
		})
	}

	parseErrorCases := []struct {
		context              string
		configYaml           string
		expectedErrorMessage string
	}{
		{
			context: "WithClusterAutoscalerEnabledForWorkers",
			configYaml: minimalValidConfigYaml + `
worker:
  clusterAutoscaler:
    minSize: 1
    maxSize: 2
`,
			expectedErrorMessage: "cluster-autoscaler support can't be enabled for a main cluster",
		},
		{
			context: "WithInvalidTaint",
			configYaml: minimalValidConfigYaml + `
experimental:
  taints:
    - key: foo
      value: bar
      effect: UnknownEffect
`,
			expectedErrorMessage: "Effect must be NoSchdule or PreferNoSchedule, but was UnknownEffect",
		},
		{
			context: "WithVpcIdAndVPCCIDRSpecified",
			configYaml: minimalValidConfigYaml + `
vpcId: vpc-1a2b3c4d
# vpcCIDR (10.1.0.0/16) does not contain instanceCIDR (10.0.1.0/24)
vpcCIDR: "10.1.0.0/16"
`,
		},
		{
			context: "WithRouteTableIdSpecified",
			configYaml: minimalValidConfigYaml + `
# vpcId must be specified if routeTableId is specified
routeTableId: rtb-1a2b3c4d
`,
		},
		{
			context: "WithWorkerSecurityGroupIds",
			configYaml: minimalValidConfigYaml + `
workerSecurityGroupIds:
  - sg-12345678
  - sg-abcdefab
  - sg-23456789
  - sg-bcdefabc
  - sg-34567890
`,
			expectedErrorMessage: "number of user provided security groups must be less than or equal to 4 but was 5",
		},
		{
			context: "WithWorkerAndLBSecurityGroupIds",
			configYaml: minimalValidConfigYaml + `
workerSecurityGroupIds:
  - sg-12345678
  - sg-abcdefab
  - sg-23456789
experimental:
  loadBalancer:
    enabled: true
    securityGroupIds:
      - sg-bcdefabc
      - sg-34567890
`,
			expectedErrorMessage: "number of user provided security groups must be less than or equal to 4 but was 5",
		},
	}

	for _, invalidCase := range parseErrorCases {
		t.Run(invalidCase.context, func(t *testing.T) {
			configBytes := invalidCase.configYaml
			providedConfig, err := config.ClusterFromBytes([]byte(configBytes))
			if err == nil {
				t.Errorf("expected to fail parsing config %s: %v", configBytes, providedConfig)
				t.FailNow()
			}

			errorMsg := fmt.Sprintf("%v", err)
			if !strings.Contains(errorMsg, invalidCase.expectedErrorMessage) {
				t.Errorf(`expected "%s" to be contained in the errror message : %s`, invalidCase.expectedErrorMessage, errorMsg)
			}
		})
	}
}
