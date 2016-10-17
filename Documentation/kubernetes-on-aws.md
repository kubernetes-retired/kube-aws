# Kubernetes on AWS

Deploy a fully-functional Kubernetes cluster using AWS CloudFormation.
Your cluster will be configured to use AWS features to enhance Kubernetes.
For example, Kubernetes may automatically provision an Elastic Load Balancer for each Kubernetes Service.
At CoreOS, we use the [kube-aws](https://github.com/coreos/coreos-kubernetes/releases) CLI tool to automate cluster deployment to AWS.

After completing this guide, a deployer will be able to interact with the Kubernetes API from their workstation using the `kubectl` CLI tool.

Each of the steps will cover:

* Step 1: Configure (this document)
  * Download the kube-aws CloudFormation generator
  * Define account and cluster settings
* [Step 2: Render][aws-step-2]
  * Compile a re-usable CloudFormation template for the cluster
  * Optionally adjust template configuration
  * Validate the rendered CloudFormation stack
* [Step 3: Launch][aws-step-3]
  * Create the CloudFormation stack and start our EC2 machines
  * Set up CLI access to the new cluster

Let's get started.

## Download kube-aws

Import the [CoreOS Application Signing Public Key](https://coreos.com/security/app-signing-key/):

```sh
gpg2 --keyserver pgp.mit.edu --recv-key FC8A365E
```

Validate the key fingerprint:

```sh
gpg2 --fingerprint FC8A365E
```
The correct key fingerprint is `18AD 5014 C99E F7E3 BA5F  6CE9 50BD D3E0 FC8A 365E`

Go to the [releases](https://github.com/coreos/coreos-kubernetes/releases) and download the latest release tarball and detached signature (.sig) for your architecture.

Validate the tarball's GPG signature:

```sh
PLATFORM=linux-amd64
# Or
PLATFORM=darwin-amd64

gpg2 --verify kube-aws-${PLATFORM}.tar.gz.sig kube-aws-${PLATFORM}.tar.gz
```
Extract the binary:

```sh
tar zxvf kube-aws-${PLATFORM}.tar.gz
```

Add kube-aws to your path:

```sh
mv ${PLATFORM}/kube-aws /usr/local/bin
```

## Configure AWS credentials

Configure your local workstation with AWS credentials using one of the following methods:

### Method 1: Configure command

Provide the values of your AWS access and secret keys, and optionally default region and output format:

```sh
$ aws configure
AWS Access Key ID [None]: AKID1234567890
AWS Secret Access Key [None]: MY-SECRET-KEY
Default region name [None]: us-west-2
Default output format [None]: text
```

### Method 2: Config file

Write your credentials into the file `~/.aws/credentials` using the following template:

```
[default]
aws_access_key_id = AKID1234567890
aws_secret_access_key = MY-SECRET-KEY
```

## Test Credentials

Test that your credentials work by describing any instances you may already have running on your account:

```sh
$ aws ec2 describe-instances
```

<div class="co-m-docs-next-step">
  <p><strong>Did you download kube-aws?</strong></p>
  <p><strong>Did your credentials work?</strong> We will use the AWS CLI in the next step.</p>
  <a href="kubernetes-on-aws-render.md" class="btn btn-primary btn-icon-right"  data-category="Docs Next" data-event="Kubernetes: AWS Render">Yes, ready to configure my cluster options</a>
  <a href="https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html" class="btn btn-default btn-icon-right"  data-category="Docs Next" data-event="Configure AWS CLI">No, I need more info on the AWS CLI</a>
</div>

[aws-step-1]: kubernetes-on-aws.md
[aws-step-2]: kubernetes-on-aws-render.md
[aws-step-3]: kubernetes-on-aws-launch.md
