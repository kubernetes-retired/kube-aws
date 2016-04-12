package cluster

import (
	"bytes"
	"errors"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/config"
)

// set by build script
var VERSION = "UNKNOWN"

type ClusterInfo struct {
	Name         string
	ControllerIP string
}

func (c *ClusterInfo) String() string {
	buf := new(bytes.Buffer)
	w := new(tabwriter.Writer)
	w.Init(buf, 0, 8, 0, '\t', 0)

	fmt.Fprintf(w, "Cluster Name:\t%s\n", c.Name)
	fmt.Fprintf(w, "Controller IP:\t%s\n", c.ControllerIP)

	w.Flush()
	return buf.String()
}

func New(cfg *config.Cluster, awsDebug bool) *Cluster {
	awsConfig := aws.NewConfig().
		WithRegion(cfg.Region).
		WithCredentialsChainVerboseErrors(true)

	if awsDebug {
		awsConfig = awsConfig.WithLogLevel(aws.LogDebug)
	}

	return &Cluster{
		Cluster: *cfg,
		session: session.New(awsConfig),
	}
}

type Cluster struct {
	config.Cluster
	session *session.Session
}

func (c *Cluster) ValidateStack(stackBody string) (string, error) {
	validateInput := cloudformation.ValidateTemplateInput{
		TemplateBody: &stackBody,
	}

	cfSvc := cloudformation.New(c.session)
	validationReport, err := cfSvc.ValidateTemplate(&validateInput)
	if err != nil {
		return "", fmt.Errorf("invalid cloudformation stack: %v", err)
	}

	return validationReport.String(), nil
}

type ec2Service interface {
	DescribeVpcs(*ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error)
	DescribeSubnets(*ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error)
	DescribeKeyPairs(*ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error)
}

func (c *Cluster) validateExistingVPCState(ec2Svc ec2Service) error {
	if c.VPCID == "" {
		//The VPC will be created. No existing state to validate
		return nil
	}

	describeVpcsInput := ec2.DescribeVpcsInput{
		VpcIds: []*string{aws.String(c.VPCID)},
	}
	vpcOutput, err := ec2Svc.DescribeVpcs(&describeVpcsInput)
	if err != nil {
		return fmt.Errorf("error describing existing vpc: %v", err)
	}
	if len(vpcOutput.Vpcs) == 0 {
		return fmt.Errorf("could not find vpc %s in region %s", c.VPCID, c.Region)
	}
	if len(vpcOutput.Vpcs) > 1 {
		//Theoretically this should never happen. If it does, we probably want to know.
		return fmt.Errorf("found more than one vpc with id %s. this is NOT NORMAL.", c.VPCID)
	}

	existingVPC := vpcOutput.Vpcs[0]

	if *existingVPC.CidrBlock != c.VPCCIDR {
		//If this is the case, our network config validation cannot be trusted and we must abort
		return fmt.Errorf("configured vpcCidr (%s) does not match actual existing vpc cidr (%s)", c.VPCCIDR, *existingVPC.CidrBlock)
	}

	describeSubnetsInput := ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{existingVPC.VpcId},
			},
		},
	}

	subnetOutput, err := ec2Svc.DescribeSubnets(&describeSubnetsInput)
	if err != nil {
		return fmt.Errorf("error describing subnets for vpc: %v", err)
	}

	subnetCIDRS := make([]string, len(subnetOutput.Subnets))
	for i, existingSubnet := range subnetOutput.Subnets {
		subnetCIDRS[i] = *existingSubnet.CidrBlock
	}

	if err := c.ValidateExistingVPC(*existingVPC.CidrBlock, subnetCIDRS); err != nil {
		return fmt.Errorf("error validating existing VPC: %v", err)
	}

	return nil
}

func (c *Cluster) Create(stackBody string) error {
	ec2Svc := ec2.New(c.session)

	if err := c.validateKeyPair(ec2Svc); err != nil {
		return err
	}

	if err := c.validateExistingVPCState(ec2Svc); err != nil {
		return err
	}

	cfSvc := cloudformation.New(c.session)
	creq := &cloudformation.CreateStackInput{
		StackName:    aws.String(c.ClusterName),
		OnFailure:    aws.String("DO_NOTHING"),
		Capabilities: []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		TemplateBody: &stackBody,
	}

	resp, err := cfSvc.CreateStack(creq)
	if err != nil {
		return err
	}

	req := cloudformation.DescribeStacksInput{
		StackName: resp.StackId,
	}
	for {
		resp, err := cfSvc.DescribeStacks(&req)
		if err != nil {
			return err
		}
		if len(resp.Stacks) == 0 {
			return fmt.Errorf("stack not found")
		}
		statusString := aws.StringValue(resp.Stacks[0].StackStatus)
		switch statusString {
		case cloudformation.ResourceStatusCreateComplete:
			return nil
		case cloudformation.ResourceStatusCreateFailed:
			errMsg := fmt.Sprintf(
				"Stack creation failed: %s : %s",
				statusString,
				aws.StringValue(resp.Stacks[0].StackStatusReason),
			)
			return errors.New(errMsg)
		case cloudformation.ResourceStatusCreateInProgress:
			time.Sleep(3 * time.Second)
			continue
		default:
			return fmt.Errorf("unexpected stack status: %s", statusString)
		}
	}
}

func (c *Cluster) Update(stackBody string) (string, error) {
	cfSvc := cloudformation.New(c.session)
	input := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		StackName:    aws.String(c.ClusterName),
		TemplateBody: &stackBody,
	}

	updateOutput, err := cfSvc.UpdateStack(input)
	if err != nil {
		return "", fmt.Errorf("error updating cloudformation stack: %v", err)
	}
	req := cloudformation.DescribeStacksInput{
		StackName: updateOutput.StackId,
	}
	for {
		resp, err := cfSvc.DescribeStacks(&req)
		if err != nil {
			return "", err
		}
		if len(resp.Stacks) == 0 {
			return "", fmt.Errorf("stack not found")
		}
		statusString := aws.StringValue(resp.Stacks[0].StackStatus)
		switch statusString {
		case cloudformation.ResourceStatusUpdateComplete:
			return updateOutput.String(), nil
		case cloudformation.ResourceStatusUpdateFailed, cloudformation.StackStatusUpdateRollbackComplete, cloudformation.StackStatusUpdateRollbackFailed:
			errMsg := fmt.Sprintf("Stack status: %s : %s", statusString, aws.StringValue(resp.Stacks[0].StackStatusReason))
			return "", errors.New(errMsg)
		case cloudformation.ResourceStatusUpdateInProgress:
			time.Sleep(3 * time.Second)
			continue
		default:
			return "", fmt.Errorf("unexpected stack status: %s", statusString)
		}
	}
}

func (c *Cluster) Info() (*ClusterInfo, error) {
	resources := make([]cloudformation.StackResourceSummary, 0)
	req := cloudformation.ListStackResourcesInput{
		StackName: aws.String(c.ClusterName),
	}
	cfSvc := cloudformation.New(c.session)
	for {
		resp, err := cfSvc.ListStackResources(&req)
		if err != nil {
			return nil, err
		}
		for _, s := range resp.StackResourceSummaries {
			resources = append(resources, *s)
		}
		req.NextToken = resp.NextToken
		if aws.StringValue(req.NextToken) == "" {
			break
		}
	}

	var info ClusterInfo
	for _, r := range resources {
		switch aws.StringValue(r.LogicalResourceId) {
		case "EIPController":
			if r.PhysicalResourceId != nil {
				info.ControllerIP = *r.PhysicalResourceId
			} else {
				return nil, fmt.Errorf("unable to get public IP of controller instance")
			}
		}
	}

	return &info, nil
}

func (c *Cluster) Destroy() error {
	cfSvc := cloudformation.New(c.session)
	dreq := &cloudformation.DeleteStackInput{
		StackName: aws.String(c.ClusterName),
	}
	_, err := cfSvc.DeleteStack(dreq)
	return err
}

func (c *Cluster) validateKeyPair(ec2Svc ec2Service) error {
	_, err := ec2Svc.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{
		KeyNames: []*string{aws.String(c.KeyName)},
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "InvalidKeyPair.NotFound" {
				return fmt.Errorf("Key %s does not exist.", c.KeyName)
			}
		}
		return err
	}

	return nil
}
