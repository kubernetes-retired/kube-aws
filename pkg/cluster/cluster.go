package cluster

import (
	"bytes"
	"errors"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"

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
	awsConfig := aws.NewConfig()
	awsConfig = awsConfig.WithRegion(cfg.Region)
	if awsDebug {
		awsConfig = awsConfig.WithLogLevel(aws.LogDebug)
	}

	return &Cluster{
		clusterName: aws.String(cfg.ClusterName),
		svc:         cloudformation.New(session.New(awsConfig)),
	}
}

type Cluster struct {
	clusterName *string
	svc         *cloudformation.CloudFormation
}

func (c *Cluster) ValidateStack(stackBody string) (string, error) {
	input := &cloudformation.ValidateTemplateInput{
		TemplateBody: &stackBody,
	}

	validationReport, err := c.svc.ValidateTemplate(input)
	if err != nil {
		return "", fmt.Errorf("Invalid cloudformation stack: %v", err)
	}

	return validationReport.String(), err
}

func (c *Cluster) Create(stackBody string) error {
	creq := &cloudformation.CreateStackInput{
		StackName:    c.clusterName,
		OnFailure:    aws.String("DO_NOTHING"),
		Capabilities: []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		TemplateBody: &stackBody,
	}

	resp, err := c.svc.CreateStack(creq)
	if err != nil {
		return err
	}

	req := cloudformation.DescribeStacksInput{
		StackName: resp.StackId,
	}
	for {
		resp, err := c.svc.DescribeStacks(&req)
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
			errMsg := fmt.Sprintf("Stack creation failed: %s : %s", statusString, aws.StringValue(resp.Stacks[0].StackStatusReason))
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
	input := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		StackName:    c.clusterName,
		TemplateBody: &stackBody,
	}

	updateOutput, err := c.svc.UpdateStack(input)
	if err != nil {
		return "", fmt.Errorf("error updating cloudformation stack: %v", err)
	}
	req := cloudformation.DescribeStacksInput{
		StackName: updateOutput.StackId,
	}
	for {
		resp, err := c.svc.DescribeStacks(&req)
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
		StackName: c.clusterName,
	}
	for {
		resp, err := c.svc.ListStackResources(&req)
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
	dreq := &cloudformation.DeleteStackInput{
		StackName: c.clusterName,
	}
	_, err := c.svc.DeleteStack(dreq)
	return err
}
