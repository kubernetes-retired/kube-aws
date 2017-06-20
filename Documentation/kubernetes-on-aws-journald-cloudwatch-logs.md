# Journald logging to AWS CloudWatch

A service has been introduced which runs a dockerised image of *journald-cloudwatch-logs*. This service forwards journald logs to AWS CloudWatch to a LogGroup with the name of .ClusterName and is run on all nodes (Etcds, Controllers and Workers).

*journald-cloudwatch-logs* is a goLang project https://github.com/saymedia/journald-cloudwatch-logs.

The default docker image *[jollinshead/journald-cloudwatch-logs](https://hub.docker.com/r/jollinshead/journald-cloudwatch-logs/)* is a wrapper around the go binary of *journald-cloudwatch-logs*.

This feature is disabled by default and configurable in cluster.yaml:

```
cloudWatchLogging:
 enabled: false
 imageWithTag: jollinshead/journald-cloudwatch-logs:0.1
 retentionInDays: 7
```