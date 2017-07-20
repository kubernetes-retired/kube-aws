package cfnstack

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kubernetes-incubator/kube-aws/model"
	"strings"
	"time"
)

type Provisioner struct {
	stackName       string
	stackTags       map[string]string
	stackPolicyBody string
	session         *session.Session
	s3URI           string
	region          model.Region
}

func NewProvisioner(name string, stackTags map[string]string, s3URI string, region model.Region, stackPolicyBody string, session *session.Session) *Provisioner {
	return &Provisioner{
		stackName:       name,
		stackTags:       stackTags,
		stackPolicyBody: stackPolicyBody,
		session:         session,
		s3URI:           s3URI,
		region:          region,
	}
}

func (c *Provisioner) uploadAsset(s3Svc S3ObjectPutterService, asset model.Asset) error {
	bucket := asset.Bucket
	key := asset.Key
	content := asset.Content
	contentLength := int64(len(content))
	body := strings.NewReader(content)

	_, err := s3Svc.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(contentLength),
		ContentType:   aws.String("application/json"),
	})

	return err
}

func (c *Provisioner) UploadAssets(s3Svc S3ObjectPutterService, assets Assets) error {
	for _, a := range assets.AsMap() {
		err := c.uploadAsset(s3Svc, a)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Provisioner) EstimateTemplateCost(cfSvc CRUDService, body string, parameters []*cloudformation.Parameter) (*cloudformation.EstimateTemplateCostOutput, error) {

	input := cloudformation.EstimateTemplateCostInput{
		TemplateBody: &body,
		Parameters:   parameters,
	}
	templateCost, err := cfSvc.EstimateTemplateCost(&input)
	return templateCost, err
}

func (c *Provisioner) CreateStackAtURLAndWait(cfSvc CRUDService, templateURL string) error {
	resp, err := c.createStackFromTemplateURL(cfSvc, templateURL)
	if err != nil {
		return err
	}
	return c.waitUntilStackGetsCreated(cfSvc, resp)
}

func (c *Provisioner) waitUntilStackGetsCreated(cfSvc CRUDService, resp *cloudformation.CreateStackOutput) error {
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
			errMsg = errMsg + "\n\nPrinting the most recent failed stack events:\n"

			stackEventsOutput, err := cfSvc.DescribeStackEvents(
				&cloudformation.DescribeStackEventsInput{
					StackName: resp.Stacks[0].StackName,
				})
			if err != nil {
				return err
			}
			errMsg = errMsg + strings.Join(StackEventErrMsgs(stackEventsOutput.StackEvents), "\n")
			return errors.New(errMsg)
		case cloudformation.ResourceStatusCreateInProgress:
			time.Sleep(3 * time.Second)
			continue
		default:
			return fmt.Errorf("unexpected stack status: %s", statusString)
		}
	}
}

func (c *Provisioner) baseCreateStackInput() *cloudformation.CreateStackInput {
	var tags []*cloudformation.Tag
	for k, v := range c.stackTags {
		key := k
		value := v
		tags = append(tags, &cloudformation.Tag{Key: &key, Value: &value})
	}

	return &cloudformation.CreateStackInput{
		StackName:       aws.String(c.stackName),
		OnFailure:       aws.String(cloudformation.OnFailureDoNothing),
		Capabilities:    []*string{aws.String(cloudformation.CapabilityCapabilityIam), aws.String(cloudformation.CapabilityCapabilityNamedIam)},
		Tags:            tags,
		StackPolicyBody: aws.String(c.stackPolicyBody),
	}
}

func (c *Provisioner) createStackFromTemplateURL(cfSvc CreationService, stackTemplateURL string) (*cloudformation.CreateStackOutput, error) {
	input := c.baseCreateStackInput()
	input.TemplateURL = &stackTemplateURL
	return cfSvc.CreateStack(input)
}

func (c *Provisioner) baseUpdateStackInput() *cloudformation.UpdateStackInput {
	return &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String(cloudformation.CapabilityCapabilityIam), aws.String(cloudformation.CapabilityCapabilityNamedIam)},
		StackName:    aws.String(c.stackName),
	}
}

func (c *Provisioner) updateStackWithTemplateURL(cfSvc UpdateService, templateURL string) (*cloudformation.UpdateStackOutput, error) {
	input := c.baseUpdateStackInput()
	input.TemplateURL = aws.String(templateURL)
	return cfSvc.UpdateStack(input)
}

func (c *Provisioner) UpdateStackAtURLAndWait(cfSvc CRUDService, templateURL string) (string, error) {
	updateOutput, err := c.updateStackWithTemplateURL(cfSvc, templateURL)
	if err != nil {
		return "", fmt.Errorf("error updating cloudformation stack: %v", err)
	}
	return c.waitUntilStackGetsUpdated(cfSvc, updateOutput)
}

func (c *Provisioner) waitUntilStackGetsUpdated(cfSvc CRUDService, updateOutput *cloudformation.UpdateStackOutput) (string, error) {
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
		case cloudformation.ResourceStatusUpdateInProgress, cloudformation.StackStatusUpdateCompleteCleanupInProgress:
			time.Sleep(3 * time.Second)
			continue
		default:
			return "", fmt.Errorf("unexpected stack status: %s", statusString)
		}
	}
}

func (c *Provisioner) ValidateStackAtURL(templateURL string) (string, error) {
	if templateURL == "" {
		return "", errors.New("[bug] ValidateStackAtURL: templateURL must not be nil")
	}

	validateInput := cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String(templateURL),
	}

	cfSvc := cloudformation.New(c.session)
	validationReport, err := cfSvc.ValidateTemplate(&validateInput)
	if err != nil {
		return "", fmt.Errorf("invalid cloudformation stack: %v", err)
	}

	return validationReport.String(), nil
}

type Destroyer struct {
	stackName string
	session   *session.Session
}

func NewDestroyer(stackName string, session *session.Session) *Destroyer {
	return &Destroyer{
		stackName: stackName,
		session:   session,
	}
}

func (c *Destroyer) Destroy() error {
	cfSvc := cloudformation.New(c.session)
	dreq := &cloudformation.DeleteStackInput{
		StackName: aws.String(c.stackName),
	}
	_, err := cfSvc.DeleteStack(dreq)
	return err
}

func (c *Provisioner) StreamCloudFormationNested(quit chan bool, stackId string, startTime time.Time) error {
	cflSwf := cloudformation.New(c.session)
	var eventId, nextEventId string
	nestedStacks := make(map[string]bool)
	nestedQuit := make(chan bool)
	defer func() { nestedQuit <- true }()
	const (
		STATE_INIT = iota
		STATE_FIRST_PAGE
		STATE_DEFAULT
	)
	state := STATE_INIT
	for {
		select {
		case <-quit:
			return nil
		default:
			outputMessage := ""
			dseInput := cloudformation.DescribeStackEventsInput{
				StackName: &stackId,
			}
			err := cflSwf.DescribeStackEventsPages(&dseInput,
				func(page *cloudformation.DescribeStackEventsOutput, lastPage bool) bool {

					if len(page.StackEvents) < 1 {
						return false
					}

					switch state {
					case STATE_INIT:
						eventId = *page.StackEvents[0].EventId
						nextEventId = eventId
						state = STATE_DEFAULT
						return false

					case STATE_FIRST_PAGE:
						nextEventId = *page.StackEvents[0].EventId
						state = STATE_DEFAULT
						fallthrough

					case STATE_DEFAULT:
						for _, event := range page.StackEvents {
							if !startTime.IsZero() && startTime.Before(*event.Timestamp) ||
								startTime.IsZero() && strings.Compare(*event.EventId, eventId) != 0 {
								outputMessage = fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t\"%s\"\n",
									(event.Timestamp).String(), *event.StackName, *event.ResourceType, *event.LogicalResourceId, *event.ResourceStatus, *event.ResourceStatusReason) + outputMessage
								if strings.Compare(*event.ResourceType, "AWS::CloudFormation::Stack") == 0 &&
									strings.Compare(*event.PhysicalResourceId, *event.StackId) != 0 &&
									!nestedStacks[*event.PhysicalResourceId] {
									nestedStacks[*event.PhysicalResourceId] = true
									go c.StreamCloudFormationNested(nestedQuit, *event.PhysicalResourceId, *event.Timestamp)
								}
							} else {
								if len(outputMessage) > 0 {
									fmt.Print(outputMessage)
								}
								if !startTime.IsZero() {
									startTime = *new(time.Time)
								}
								eventId = nextEventId
								return false
							}
						}
					}

					return true
				})
			if err != nil {
				fmt.Errorf("failed to get CloudFormation events")
				continue
			}
			if state != STATE_INIT {
				state = STATE_FIRST_PAGE
			}
		}
		time.Sleep(time.Second)
	}
}
