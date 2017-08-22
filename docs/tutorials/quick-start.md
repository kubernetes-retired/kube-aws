# Quick Start

Deploy a fully-functional Kubernetes cluster using AWS CloudFormation.

Your cluster will be configured to use AWS features to enhance Kubernetes.

For example, Kubernetes may automatically provision an Elastic Load Balancer for each Kubernetes Service.

After completing this guide, a deployer will be able to interact with the Kubernetes API from their workstation using the `kubectl` CLI tool.

# Pre-requisites {#pre-requisites}

If you're deploying a cluster with kube-aws:

* [EC2 instances whose types are larger than or equal to `t2.medium` should be chosen for the cluster to work reliably](https://github.com/kubernetes-incubator/kube-aws/issues/138)
* [At least 3 etcd, 2 controller, 2 worker nodes are required to achieve high availability](https://github.com/kubernetes-incubator/kube-aws/issues/138#issuecomment-266432162)
* If you wish to deploy to an existing VPC, there is additional information on [Use An Existing VPC](/advanced-topics/use-an-existing-vpc.md) not covered by this getting started guide.

Once you understand the pre-requisites, you are ready to launch your first Kubernetes cluster.

# Step 1: Configure {#step1}

Step 1 will cover:

* Downloading kube-aws
* Defining account and cluster settings

## Download kube-aws

Go to the [releases](https://github.com/kubernetes-incubator/kube-aws/releases) and download the latest release tarball for your architecture. Extract the binary:

```
tar zxvf kube-aws-${PLATFORM}.tar.gz
```

Add kube-aws to your path:

```
mv ${PLATFORM}/kube-aws /usr/local/bin
```

## Configure AWS credentials

Configure your local workstation with AWS credentials using one of the following methods:

**Method 1: Configure command**

Provide the values of your AWS access and secret keys, and optionally default region and output format:

```
$ aws configure
AWS Access Key ID [None]: AKID1234567890
AWS Secret Access Key [None]: MY-SECRET-KEY
Default region name [None]: us-west-2
Default output format [None]: text
```

**Method 2: Config file**

Write your credentials into the file \`~/.aws/credentials\` using the following template:

```
[default]
aws_access_key_id = AKID1234567890
aws_secret_access_key = MY-SECRET-KEY
```

**Method 3: Environment variables**

Provide AWS credentials to kube-aws by exporting the following environment variables:

```
export AWS_ACCESS_KEY_ID=AKID1234567890
export AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

## Test Credentials

Test that your credentials work by describing any instances you may already have running on your account:

```
$ aws ec2 describe-instances
```

# Step 2: Render

Step 2 will cover:

* Compiling a re-usable CloudFormation template for the cluster
* Optionally adjust template configuration
* Validate the rendered CloudFormation stack

# Step 3: Launch

Step 3 will cover:

* Create the CloudFormation stack and start our EC2 machines
* Set up CLI access to the new cluster

# Step 4: Update

* Update the CloudFormation stack

# Step 5: Add Node Pool

Step 5 will cover:

* Create the additional pool of worker nodes
* Adjust template configuration for each pool of worker nodes
* Required to support [cluster-autoscaler](https://github.com/kubernetes/contrib/tree/master/cluster-autoscaler)

# Step 6: Configure Add-ons

Step 6 will cover:

* Configure various Kubernetes add-ons

# Step 7: Destroy

Step 7 will cover:

* Tearing down the cluster

Let's get started.

