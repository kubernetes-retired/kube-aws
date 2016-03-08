# Kubernetes on AWS

This is the source of the `kube-aws` tool and the installation artifacts used by the official Kubernetes on AWS documentation.
View the full instructions at https://coreos.com/kubernetes/docs/latest/kubernetes-on-aws.html.

## Development

### Download pre-built binary

```sh
wget https://<binary-url>
# check checksum
chmod +x ./kube-aws
sudo mv kube-aws /usr/bin/
```

### Build

Run the `./build` script to compile `kube-aws` locally.

This depends on having:
* golang >= 1.5

The compiled binary will be available at `./bin/kube-aws`.

## Initialize an asset directory
```sh
$ mkdir my-cluster
$ cd ./my-cluster
$ kube-aws init --cluster-name=my-cluster-name --external-dns-name=my-cluster-endpoint --region=us-west-1 --availability-zone=us-west-1c --key-name=key-pair-name
```

There will now be a ./cluster.yaml file in the asset directory.

## Render contents of the asset directory

```sh
$ kube-aws render
```
This generates the default set of cluster assets in your asset directory. These assets are templates and credentials that are used to create, update and interact with your Kubernetes cluster.

You can now customize your cluster by editing asset files:

* **./cluster.yaml**

  This is the configuration file for your cluster. It contains the configuration parameters that are templated into your userdata and cloudformation stack.

* **./cloud-config/**

  * `./cloud-config-worker`
  * `./cloud-config-controller`

  This directory contains the [cloud-init](https://github.com/coreos/coreos-cloudinit) cloud-config userdata files. The CoreOS operating system supports automated provisioning via cloud-config files, which describe the various files, scripts and systemd actions necessary to produce a working cluster machine. These files are templated with your cluster configuration parameters and embedded into the cloudformation stack template.

* **./stack-template.json**

  This file describes the [AWS cloudformation](https://aws.amazon.com/cloudformation/) stack which encompasses all the AWS resources associated with your cluster. This JSON document is temlated with configuration parameters, we well as the encoded userdata files.

* **credentials/**

  This directory contains the **unencrypted** TLS assets for your cluster, along with a pre-configured `kubeconfig` file which provides access to your cluster api via kubectl.

You can also now check the `./my-cluster` asset directory into version control if you desire. The contents of this directory are your reproducible cluster assets. Please take care not to commit the `./my-cluster/credentials` directory, as it contains your TLS secrets. If you're using git, the `credentials` directory will already be ignored for you.

## Validate your cluster assets

The `validate` command check the validity of the cloud-config userdata files and the cloudformation stack description.

```sh
$ kube-aws validate
```

## Create a cluster from asset directory

```sh
$ kube-aws up
```

This command can take a while.

## Access the cluster

```sh
$ kubectl --kubeconfig=./credentials/kubeconfig get nodes
```

It can take some time after `kube-aws up` completes before the cluster is available. Until then, you will have a `connection refused` error.

### Useful Resources

The following links can be useful for development:

- [AWS CloudFormation resource types](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-template-resource-type-ref.html)

## Contributing

Submit a PR to this repository, following the [contributors guide](../../CONTRIBUTING.md).
The documentation is published from [this source](../../Documentation/kubernetes-on-aws.md).


