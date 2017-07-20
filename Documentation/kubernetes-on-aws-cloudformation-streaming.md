# CloudFormation Streaming

CouldFormation stack events can be streamed to the console (stdout) during a `kube-aws up` or `kube-aws update` operation.
The format of the messages are:
```
Timestamp   StackName   ResourceType    LogicalResourceId   ResourceStatus
```
For example:
```
2017-07-19 09:09:06.174 +0000 UTC	my-cluster  AWS::CloudFormation::Stack	Controlplane	UPDATE_IN_PROGRESS
2017-07-19 09:09:07.008 +0000 UTC	my-cluster  AWS::CloudFormation::Stack	Controlplane	UPDATE_COMPLETE
2017-07-19 09:09:10.593 +0000 UTC	my-cluster  AWS::CloudFormation::Stack	Nodepoola	UPDATE_IN_PROGRESS
2017-07-19 09:09:16.635 +0000 UTC	my-cluster-Nodepoola-1XBQEN77K5CCE  AWS::AutoScaling::AutoScalingGroup	Workers	UPDATE_IN_PROGRESS
```

This feature is enabled by default and configurable in cluster.yaml:

```
cloudFormationStreaming: true
```
