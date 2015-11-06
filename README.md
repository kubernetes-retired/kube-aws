# Kubernetes on AWS

This is the source of the `kube-aws` tool and the installation artifacts used by the official Kubernetes on AWS documentation.
View the full instructions at https://coreos.com/kubernetes/docs/latest/kubernetes-on-aws.html.

## Development

### Build

Run the `./build` script to compile `kube-aws` locally.
This depends on having golang available on your workstation.
The compiled binary will be available at `./bin/kube-aws`.

### Custom Kubernetes Manifests

You may deploy a cluster using a custom CloudFormation template, Kubernetes manifests and install scripts using the `artifactURL` option in your cluster config.

For example, you might upload a modified set of manifests to a custom S3 bucket (making the files publicly-readable) using the following commands:

```
$ kube-aws render --output=artifacts/template.json
$ aws s3 cp --recursive --acl=public-read artifacts/ s3://<bucket>/
```

Then, simply create a cluster using `artifactURL: https://<bucket>.s3.amazonaws.com`.

### Useful Resources

The following links can be useful for development:

- [AWS CloudFormation resource types](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-template-resource-type-ref.html)

## Contributing

Submit a PR to this repository, following the [contributors guide](../../CONTRIBUTING.md).
The documentation is published from [this source](../../Documentation/kubernetes-on-aws.md).


