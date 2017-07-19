# CloudFormation Streaming

CouldFormation stack events can be streamed to the console (stdout) during a `kube-aws up` or `kube-aws update` operation.
The format of the messages are:
```
Timestamp   ResourceType    LogicalResourceId   ResourceStatus  (StackName)
```
For example:
```
2017-07-19 09:09:06.174 +0000 UTC	AWS::CloudFormation::Stack	Controlplane	UPDATE_IN_PROGRESS	(my-cluster)
2017-07-19 09:09:07.008 +0000 UTC	AWS::CloudFormation::Stack	Controlplane	UPDATE_COMPLETE	(my-cluster)
2017-07-19 09:09:10.593 +0000 UTC	AWS::CloudFormation::Stack	Nodepoola	UPDATE_IN_PROGRESS	(my-cluster)
2017-07-19 09:09:16.635 +0000 UTC	AWS::AutoScaling::AutoScalingGroup	Workers	UPDATE_IN_PROGRESS	(my-cluster-Nodepoola-1XBQEN77K5CCE)
```
NOTE: While events are likely to stream in order by timestamp, it is not guaranteed.

This feature is enabled by default and configurable in cluster.yaml:

```
cloudFormationStreaming: true
```
