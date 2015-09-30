package cluster

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func createStackAndWait(svc *cloudformation.CloudFormation, name, templateURL string, parameters []*cloudformation.Parameter) error {
	creq := &cloudformation.CreateStackInput{
		StackName:    aws.String(name),
		OnFailure:    aws.String("DO_NOTHING"),
		Capabilities: []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		Parameters:   parameters,
		TemplateURL:  aws.String(templateURL),
	}

	resp, err := svc.CreateStack(creq)
	if err != nil {
		return err
	}

	if err := waitForStackCreateComplete(svc, aws.StringValue(resp.StackId)); err != nil {
		return err
	}

	return nil
}

func waitForStackCreateComplete(svc *cloudformation.CloudFormation, stackID string) error {
	req := cloudformation.DescribeStacksInput{
		StackName: aws.String(stackID),
	}
	for {
		resp, err := svc.DescribeStacks(&req)
		if err != nil {
			return err
		}
		if len(resp.Stacks) == 0 {
			return fmt.Errorf("stack not found")
		}
		switch aws.StringValue(resp.Stacks[0].StackStatus) {
		case cloudformation.ResourceStatusCreateComplete:
			return nil
		case cloudformation.ResourceStatusCreateFailed:
			return errors.New(aws.StringValue(resp.Stacks[0].StackStatusReason))
		}
		time.Sleep(3 * time.Second)
	}
}

func getStackResources(svc *cloudformation.CloudFormation, stackID string) ([]cloudformation.StackResourceSummary, error) {
	resources := make([]cloudformation.StackResourceSummary, 0)
	req := cloudformation.ListStackResourcesInput{
		StackName: aws.String(stackID),
	}
	for {
		resp, err := svc.ListStackResources(&req)
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
	return resources, nil
}

func mapStackResourcesToClusterInfo(svc *ec2.EC2, resources []cloudformation.StackResourceSummary) (*ClusterInfo, error) {
	var info ClusterInfo
	for _, r := range resources {
		switch aws.StringValue(r.LogicalResourceId) {
		case resNameEIPController:
			if r.PhysicalResourceId != nil {
				info.ControllerIP = *r.PhysicalResourceId
			} else {
				return nil, fmt.Errorf("unable to get public IP of controller instance")
			}
		}
	}

	return &info, nil
}

func destroyStack(svc *cloudformation.CloudFormation, name string) error {
	dreq := &cloudformation.DeleteStackInput{
		StackName: aws.String(name),
	}
	_, err := svc.DeleteStack(dreq)
	return err
}
