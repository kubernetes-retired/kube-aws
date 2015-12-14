package cluster

import (
	"encoding/json"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/coreosutil"
)

const (
	// resource names
	resNameVPC                          = "VPC"
	resNameInternetGateway              = "InternetGateway"
	resNameVPCGatewayAttachment         = "VPCGatewayAttachment"
	resNameRouteTable                   = "RouteTable"
	resNameRouteToInternet              = "RouteToInternet"
	resNameSubnet                       = "Subnet"
	resNameSubnetRouteTableAssociation  = "SubnetRouteTableAssociation"
	resNameSecurityGroupController      = "SecurityGroupController"
	resNameSecurityGroupWorker          = "SecurityGroupWorker"
	resNameInstanceController           = "InstanceController"
	resNameEIPController                = "EIPController"
	resNameAlarmControllerRecover       = "AlarmControllerRecover"
	resNameAutoScaleWorker              = "AutoScaleWorker"
	resNameLaunchConfigurationWorker    = "LaunchConfigurationWorker"
	resNameIAMRoleController            = "IAMRoleController"
	resNameIAMInstanceProfileController = "IAMInstanceProfileController"
	resNameIAMRoleWorker                = "IAMRoleWorker"
	resNameIAMInstanceProfileWorker     = "IAMInstanceProfileWorker"

	// parameter names
	parClusterName                  = "ClusterName"
	parNameReleaseChannel           = "ReleaseChannel"
	parNameControllerInstanceType   = "ControllerInstanceType"
	parNameControllerRootVolumeSize = "ControllerRootVolumeSize"
	parNameWorkerInstanceType       = "WorkerInstanceType"
	parNameKeyName                  = "KeyName"
	parArtifactURL                  = "ArtifactURL"
	parCACert                       = "CACert"
	parAPIServerCert                = "APIServerCert"
	parAPIServerKey                 = "APIServerKey"
	parWorkerCert                   = "WorkerCert"
	parWorkerKey                    = "WorkerKey"
	parWorkerCount                  = "WorkerCount"
	parNameWorkerRootVolumeSize     = "WorkerRootVolumeSize"
	parAvailabilityZone             = "AvailabilityZone"
)

var (
	supportedChannels    = []string{"alpha", "beta"}
	tagKubernetesCluster = "KubernetesCluster"

	sgProtoTCP  = "tcp"
	sgProtoUDP  = "udp"
	sgProtoICMP = "icmp"

	sgPortMax = 65535
	sgAllIPs  = "0.0.0.0/0"
)

func newTag(key string, value interface{}) map[string]interface{} {
	return map[string]interface{}{"Key": key, "Value": value}
}

func newPropagatingTag(key string, value interface{}) map[string]interface{} {
	return map[string]interface{}{"Key": key, "Value": value, "PropagateAtLaunch": "true"}
}

func newRef(name string) map[string]interface{} {
	return map[string]interface{}{"Ref": name}
}

func newIAMPolicyStatement(action, resource string) map[string]interface{} {
	return map[string]interface{}{
		"Effect":   "Allow",
		"Action":   action,
		"Resource": resource,
	}
}

func getRegionMap() (map[string]interface{}, error) {
	regionMap := map[string]map[string]string{}

	for _, channel := range supportedChannels {
		regions, err := coreosutil.GetAMIData(channel)

		if err != nil {
			return nil, err
		}

		for region, amis := range regions {
			if region == "release_info" {
				continue
			}

			if _, ok := regionMap[region]; !ok {
				regionMap[region] = map[string]string{}
			}

			if ami, ok := amis["hvm"]; ok {
				regionMap[region][channel] = ami
			}
		}
	}

	output := map[string]interface{}{}

	for key, val := range regionMap {
		output[key] = val
	}

	return output, nil
}

func StackTemplateBody(defaultArtifactURL string) (string, error) {
	// NOTE: AWS only allows non-alphanumeric keys in the top level key
	imageID := map[string]interface{}{
		"Fn::FindInMap": []interface{}{
			"RegionMap",
			newRef("AWS::Region"),
			newRef(parNameReleaseChannel),
		},
	}

	defaultAvailabilityZone := map[string]interface{}{
		"Fn::Select": []interface{}{
			"0",
			map[string]interface{}{
				"Fn::GetAZs": newRef("AWS::Region"),
			},
		},
	}

	availabilityZone := map[string]interface{}{
		"Fn::If": []interface{}{
			"EmptyAvailabilityZone",
			defaultAvailabilityZone,
			newRef(parAvailabilityZone),
		},
	}

	res := make(map[string]interface{})

	res[resNameVPC] = map[string]interface{}{
		"Type": "AWS::EC2::VPC",
		"Properties": map[string]interface{}{
			"CidrBlock":          "10.0.0.0/16",
			"EnableDnsSupport":   true,
			"EnableDnsHostnames": true,
			"InstanceTenancy":    "default",
			"Tags": []map[string]interface{}{
				newTag(tagKubernetesCluster, newRef(parClusterName)),

				// Name required to be "kubernetes-vpc" until fixed upstream
				// https://github.com/kubernetes/kubernetes/issues/9801
				newTag("Name", "kubernetes-vpc"),
			},
		},
	}

	res[resNameInternetGateway] = map[string]interface{}{
		"Type": "AWS::EC2::InternetGateway",
		"Properties": map[string]interface{}{
			"Tags": []map[string]interface{}{
				newTag(tagKubernetesCluster, newRef(parClusterName)),
			},
		},
	}

	res[resNameVPCGatewayAttachment] = map[string]interface{}{
		"Type": "AWS::EC2::VPCGatewayAttachment",
		"Properties": map[string]interface{}{
			"InternetGatewayId": newRef(resNameInternetGateway),
			"VpcId":             newRef(resNameVPC),
		},
	}

	res[resNameRouteTable] = map[string]interface{}{
		"Type": "AWS::EC2::RouteTable",
		"Properties": map[string]interface{}{
			"VpcId": newRef(resNameVPC),
			"Tags": []map[string]interface{}{
				newTag(tagKubernetesCluster, newRef(parClusterName)),
			},
		},
	}

	res[resNameRouteToInternet] = map[string]interface{}{
		"Type": "AWS::EC2::Route",
		"Properties": map[string]interface{}{
			"DestinationCidrBlock": "0.0.0.0/0",
			"RouteTableId":         newRef(resNameRouteTable),
			"GatewayId":            newRef(resNameInternetGateway),
		},
	}

	res[resNameSubnet] = map[string]interface{}{
		"Type": "AWS::EC2::Subnet",
		"Properties": map[string]interface{}{
			"AvailabilityZone":    availabilityZone,
			"CidrBlock":           "10.0.0.0/24",
			"MapPublicIpOnLaunch": true,
			"VpcId":               newRef(resNameVPC),
			"Tags": []map[string]interface{}{
				newTag(tagKubernetesCluster, newRef(parClusterName)),
			},
		},
	}

	res[resNameSubnetRouteTableAssociation] = map[string]interface{}{
		"Type": "AWS::EC2::SubnetRouteTableAssociation",
		"Properties": map[string]interface{}{
			"RouteTableId": newRef(resNameRouteTable),
			"SubnetId":     newRef(resNameSubnet),
		},
	}

	res[resNameSecurityGroupController] = map[string]interface{}{
		"Type": "AWS::EC2::SecurityGroup",
		"Properties": map[string]interface{}{
			"GroupDescription": newRef("AWS::StackName"),
			"VpcId":            newRef(resNameVPC),
			"SecurityGroupEgress": []map[string]interface{}{
				map[string]interface{}{"IpProtocol": sgProtoTCP, "FromPort": 0, "ToPort": sgPortMax, "CidrIp": sgAllIPs},
				map[string]interface{}{"IpProtocol": sgProtoUDP, "FromPort": 0, "ToPort": sgPortMax, "CidrIp": sgAllIPs},
			},
			"SecurityGroupIngress": []map[string]interface{}{
				map[string]interface{}{"IpProtocol": sgProtoICMP, "FromPort": 3, "ToPort": -1, "CidrIp": sgAllIPs},
				map[string]interface{}{"IpProtocol": sgProtoTCP, "FromPort": 22, "ToPort": 22, "CidrIp": sgAllIPs},
				map[string]interface{}{"IpProtocol": sgProtoTCP, "FromPort": 443, "ToPort": 443, "CidrIp": sgAllIPs},
			},
			"Tags": []map[string]interface{}{
				newTag(tagKubernetesCluster, newRef(parClusterName)),
			},
		},
	}
	res[resNameSecurityGroupController+"IngressFromWorkerToEtcd"] = map[string]interface{}{
		"Type": "AWS::EC2::SecurityGroupIngress",
		"Properties": map[string]interface{}{
			"GroupId":               newRef(resNameSecurityGroupController),
			"SourceSecurityGroupId": newRef(resNameSecurityGroupWorker),
			"FromPort":              2379,
			"ToPort":                2379,
			"IpProtocol":            sgProtoTCP,
		},
	}

	res[resNameSecurityGroupWorker] = map[string]interface{}{
		"Type": "AWS::EC2::SecurityGroup",
		"Properties": map[string]interface{}{
			"GroupDescription": newRef("AWS::StackName"),
			"VpcId":            newRef(resNameVPC),
			"SecurityGroupEgress": []map[string]interface{}{
				map[string]interface{}{"IpProtocol": sgProtoTCP, "FromPort": 0, "ToPort": sgPortMax, "CidrIp": sgAllIPs},
				map[string]interface{}{"IpProtocol": sgProtoUDP, "FromPort": 0, "ToPort": sgPortMax, "CidrIp": sgAllIPs},
			},
			"SecurityGroupIngress": []map[string]interface{}{
				map[string]interface{}{"IpProtocol": sgProtoICMP, "FromPort": 3, "ToPort": -1, "CidrIp": sgAllIPs},
				map[string]interface{}{"IpProtocol": sgProtoTCP, "FromPort": 22, "ToPort": 22, "CidrIp": sgAllIPs},
			},
			"Tags": []map[string]interface{}{
				newTag(tagKubernetesCluster, newRef(parClusterName)),
			},
		},
	}
	res[resNameSecurityGroupWorker+"IngressFromWorkerToFlannel"] = map[string]interface{}{
		"Type": "AWS::EC2::SecurityGroupIngress",
		"Properties": map[string]interface{}{
			"GroupId":               newRef(resNameSecurityGroupWorker),
			"SourceSecurityGroupId": newRef(resNameSecurityGroupWorker),
			"FromPort":              8285,
			"ToPort":                8285,
			"IpProtocol":            sgProtoUDP,
		},
	}
	res[resNameSecurityGroupWorker+"IngressFromWorkerToKubeletReadOnly"] = map[string]interface{}{
		"Type": "AWS::EC2::SecurityGroupIngress",
		"Properties": map[string]interface{}{
			"GroupId":               newRef(resNameSecurityGroupWorker),
			"SourceSecurityGroupId": newRef(resNameSecurityGroupWorker),
			"FromPort":              10255,
			"ToPort":                10255,
			"IpProtocol":            sgProtoTCP,
		},
	}
	res[resNameSecurityGroupWorker+"IngressFromControllerToFlannel"] = map[string]interface{}{
		"Type": "AWS::EC2::SecurityGroupIngress",
		"Properties": map[string]interface{}{
			"GroupId":               newRef(resNameSecurityGroupWorker),
			"SourceSecurityGroupId": newRef(resNameSecurityGroupController),
			"FromPort":              8285,
			"ToPort":                8285,
			"IpProtocol":            sgProtoUDP,
		},
	}
	res[resNameSecurityGroupWorker+"IngressFromControllerTocAdvisor"] = map[string]interface{}{
		"Type": "AWS::EC2::SecurityGroupIngress",
		"Properties": map[string]interface{}{
			"GroupId":               newRef(resNameSecurityGroupWorker),
			"SourceSecurityGroupId": newRef(resNameSecurityGroupController),
			"FromPort":              4194,
			"ToPort":                4194,
			"IpProtocol":            sgProtoTCP,
		},
	}
	res[resNameSecurityGroupWorker+"IngressFromControllerToKubelet"] = map[string]interface{}{
		"Type": "AWS::EC2::SecurityGroupIngress",
		"Properties": map[string]interface{}{
			"GroupId":               newRef(resNameSecurityGroupWorker),
			"SourceSecurityGroupId": newRef(resNameSecurityGroupController),
			"FromPort":              10250,
			"ToPort":                10250,
			"IpProtocol":            sgProtoTCP,
		},
	}

	res[resNameIAMRoleController] = map[string]interface{}{
		"Type": "AWS::IAM::Role",
		"Properties": map[string]interface{}{
			"AssumeRolePolicyDocument": map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					map[string]interface{}{
						"Effect": "Allow",
						"Principal": map[string]interface{}{
							"Service": []string{"ec2.amazonaws.com"},
						},
						"Action": []string{"sts:AssumeRole"},
					},
				},
			},
			"Path": "/",
			"Policies": []map[string]interface{}{
				map[string]interface{}{
					"PolicyName": "root",
					"PolicyDocument": map[string]interface{}{
						"Version": "2012-10-17",
						"Statement": []map[string]interface{}{
							newIAMPolicyStatement("ec2:*", "*"),
							newIAMPolicyStatement("elasticloadbalancing:*", "*"),
						},
					},
				},
			},
		},
	}

	res[resNameIAMRoleWorker] = map[string]interface{}{
		"Type": "AWS::IAM::Role",
		"Properties": map[string]interface{}{
			"AssumeRolePolicyDocument": map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					map[string]interface{}{
						"Effect": "Allow",
						"Principal": map[string]interface{}{
							"Service": []string{"ec2.amazonaws.com"},
						},
						"Action": []string{"sts:AssumeRole"},
					},
				},
			},
			"Path": "/",
			"Policies": []map[string]interface{}{
				map[string]interface{}{
					"PolicyName": "root",
					"PolicyDocument": map[string]interface{}{
						"Version": "2012-10-17",
						"Statement": []map[string]interface{}{
							newIAMPolicyStatement("ec2:Describe*", "*"),
							newIAMPolicyStatement("ec2:AttachVolume", "*"),
							newIAMPolicyStatement("ec2:DetachVolume", "*"),
						},
					},
				},
			},
		},
	}

	res[resNameIAMInstanceProfileController] = map[string]interface{}{
		"Type": "AWS::IAM::InstanceProfile",
		"Properties": map[string]interface{}{
			"Path": "/",
			"Roles": []map[string]interface{}{
				newRef(resNameIAMRoleController),
			},
		},
	}

	res[resNameIAMInstanceProfileWorker] = map[string]interface{}{
		"Type": "AWS::IAM::InstanceProfile",
		"Properties": map[string]interface{}{
			"Path": "/",
			"Roles": []map[string]interface{}{
				newRef(resNameIAMRoleWorker),
			},
		},
	}

	res[resNameEIPController] = map[string]interface{}{
		"Type": "AWS::EC2::EIP",
		"Properties": map[string]interface{}{
			"InstanceId": newRef(resNameInstanceController),
			"Domain":     "vpc",
		},
	}

	res[resNameInstanceController] = map[string]interface{}{
		"Type": "AWS::EC2::Instance",
		"Properties": map[string]interface{}{
			"ImageId":      imageID,
			"InstanceType": newRef(parNameControllerInstanceType),
			"KeyName":      newRef(parNameKeyName),
			"UserData": map[string]interface{}{
				"Fn::Base64": renderTemplate(baseControllerCloudConfig),
			},
			"IamInstanceProfile": newRef(resNameIAMInstanceProfileController),
			"NetworkInterfaces": []map[string]interface{}{
				map[string]interface{}{
					"PrivateIpAddress":         "10.0.0.50",
					"AssociatePublicIpAddress": false,
					"DeleteOnTermination":      true,
					"DeviceIndex":              "0",
					"SubnetId":                 newRef(resNameSubnet),
					"GroupSet": []map[string]interface{}{
						newRef(resNameSecurityGroupController),
					},
				},
			},
			"BlockDeviceMappings": []map[string]interface{}{
				map[string]interface{}{
					"DeviceName": "/dev/xvda",
					"Ebs": map[string]interface{}{
						"VolumeSize": newRef(parNameControllerRootVolumeSize),
					},
				},
			},
			"AvailabilityZone": availabilityZone,
			"Tags": []map[string]interface{}{
				newTag(tagKubernetesCluster, newRef(parClusterName)),
				newTag("Name", "kube-aws-controller"),
			},
		},
	}

	res[resNameAlarmControllerRecover] = map[string]interface{}{
		"Type": "AWS::CloudWatch::Alarm",
		"Properties": map[string]interface{}{
			"AlarmDescription":   "Trigger a recovery when system check fails for 5 consecutive minutes.",
			"Namespace":          "AWS/EC2",
			"MetricName":         "StatusCheckFailed_System",
			"Statistic":          "Minimum",
			"Period":             "60",
			"EvaluationPeriods":  "5",
			"ComparisonOperator": "GreaterThanThreshold",
			"Threshold":          "0",
			"AlarmActions": []interface{}{
				map[string]interface{}{
					"Fn::Join": []interface{}{
						"",
						[]interface{}{
							"arn:aws:automate:",
							newRef("AWS::Region"),
							":ec2:recover",
						},
					},
				},
			},
			"Dimensions": []interface{}{
				map[string]interface{}{
					"Name":  "InstanceId",
					"Value": newRef(resNameInstanceController),
				},
			},
		},
	}

	res[resNameLaunchConfigurationWorker] = map[string]interface{}{
		"Type": "AWS::AutoScaling::LaunchConfiguration",
		"Properties": map[string]interface{}{
			"ImageId":      imageID,
			"InstanceType": newRef(parNameWorkerInstanceType),
			"KeyName":      newRef(parNameKeyName),
			"UserData": map[string]interface{}{
				"Fn::Base64": renderTemplate(baseWorkerCloudConfig),
			},
			"SecurityGroups":     []interface{}{newRef(resNameSecurityGroupWorker)},
			"IamInstanceProfile": newRef(resNameIAMInstanceProfileWorker),
			"BlockDeviceMappings": []map[string]interface{}{
				map[string]interface{}{
					"DeviceName": "/dev/xvda",
					"Ebs": map[string]interface{}{
						"VolumeSize": newRef(parNameWorkerRootVolumeSize),
					},
				},
			},
		},
	}

	res[resNameAutoScaleWorker] = map[string]interface{}{
		"Type": "AWS::AutoScaling::AutoScalingGroup",
		"Properties": map[string]interface{}{
			"AvailabilityZones":       []interface{}{availabilityZone},
			"LaunchConfigurationName": newRef(resNameLaunchConfigurationWorker),
			"DesiredCapacity":         newRef(parWorkerCount),
			"MinSize":                 newRef(parWorkerCount),
			"MaxSize":                 newRef(parWorkerCount),
			"HealthCheckGracePeriod":  600,
			"HealthCheckType":         "EC2",
			"VPCZoneIdentifier":       []interface{}{newRef(resNameSubnet)},
			"Tags": []interface{}{
				newPropagatingTag(tagKubernetesCluster, newRef(parClusterName)),
				newPropagatingTag("Name", "kube-aws-worker"),
			},
		},
	}

	par := map[string]interface{}{}

	par[parClusterName] = map[string]interface{}{
		"Type":        "String",
		"Default":     "kubernetes",
		"Description": "Name of Kubernetes cluster",
	}

	// TODO(silas): change default to stable once Kubernetes is supported
	par[parNameReleaseChannel] = map[string]interface{}{
		"Type":          "String",
		"Default":       "alpha",
		"AllowedValues": supportedChannels,
		"Description":   "CoreOS Linux release channel to use as instance operating system",
	}

	par[parNameControllerInstanceType] = map[string]interface{}{
		"Type":        "String",
		"Default":     "m3.medium",
		"Description": "EC2 instance type used for each controller instance",
	}

	par[parNameControllerRootVolumeSize] = map[string]interface{}{
		"Type":        "String",
		"Default":     "30",
		"Description": "Controller root volume size (GiB)",
	}

	par[parNameWorkerInstanceType] = map[string]interface{}{
		"Type":        "String",
		"Default":     "m3.medium",
		"Description": "EC2 instance type used for each worker instance",
	}

	par[parNameKeyName] = map[string]interface{}{
		"Type":        "String",
		"Description": "Name of SSH keypair to authorize on each instance",
	}

	par[parArtifactURL] = map[string]interface{}{
		"Type":        "String",
		"Default":     defaultArtifactURL,
		"Description": "Public location of coreos-kubernetes deployment artifacts",
	}

	par[parCACert] = map[string]interface{}{
		"Type":        "String",
		"Description": "PEM-formattd CA certificate, base64-encoded",
	}

	par[parAPIServerCert] = map[string]interface{}{
		"Type":        "String",
		"Description": "PEM-formatted kube-apiserver certificate, base64-encoded",
	}

	par[parAPIServerKey] = map[string]interface{}{
		"Type":        "String",
		"Description": "PEM-formatted kube-apiserver key, base64-encoded",
	}

	par[parWorkerCert] = map[string]interface{}{
		"Type":        "String",
		"Description": "PEM-formatted kubelet (worker) certificate, base64-encoded",
	}

	par[parWorkerKey] = map[string]interface{}{
		"Type":        "String",
		"Description": "PEM-formatted kubelet (worker) key, base64-encoded",
	}

	par[parWorkerCount] = map[string]interface{}{
		"Type":        "String",
		"Default":     "1",
		"Description": "Number of worker instances to create, may be modified later",
	}

	par[parNameWorkerRootVolumeSize] = map[string]interface{}{
		"Type":        "String",
		"Default":     "30",
		"Description": "Worker root volume size (GiB)",
	}

	par[parAvailabilityZone] = map[string]interface{}{
		"Type":        "String",
		"Default":     "",
		"Description": "Specific availability zone (optional)",
	}

	regionMap, err := getRegionMap()

	if err != nil {
		return "", err
	}

	mappings := map[string]interface{}{
		"RegionMap": regionMap,
	}

	conditions := map[string]interface{}{
		"EmptyAvailabilityZone": map[string]interface{}{
			"Fn::Equals": []interface{}{
				newRef(parAvailabilityZone),
				"",
			},
		},
	}

	tmpl := map[string]interface{}{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Description":              "kube-aws Kubernetes cluster",
		"Resources":                res,
		"Parameters":               par,
		"Mappings":                 mappings,
		"Conditions":               conditions,
	}

	t, err := json.MarshalIndent(tmpl, "", "    ")
	return string(t), err
}
