# kube-aws cluster updates

## Types of cluster update
There are two distinct categories of cluster update.

* **Parameter-level update**: Only changes to `cluster.yaml` and/or TLS assets in `credentials/` folder are reflected. To enact this type of update. Modifications to CloudFormation or cloud-config userdata templates will not be reflected. In this case, you do not have to re-render:

```sh
kube-aws up --update
```

* **Full update**: Any change (besides changes made to the etcd cluster- more on that later) will be enacted, including structural changes to CloudFormation and cloudinit templates. This is the type of upgrade that must be run on installing a new version of kube-aws, or more generally when cloudinit or CloudFormation templates are modified:

```sh
kube-aws render
git diff # view changes to rendered assets
kube-aws up --update
```

## Certificate rotation

The parameter-level update mechanism can be used to rotate in new TLS credentials:

```sh
kube-aws render --generate-credentials
kube-aws up --update
```

## the etcd caveat

There is no solution for hosting an etcd cluster in a way that is easily updateable in this fashion- so updates are automatically masked for the etcd instances. This means that, after the cluster is created, nothing about the etcd ec2 instances is allowed to be updated.

Fortunately, CoreOS update engine will take care of keeping the members of the etcd cluster up-to-date, but you as the operator will not be able to modify them after creation via the update mechanism.

In the (near) future, etcd will be hosted on Kubernetes and this problem will no longer be relevant. Rather than concocting overly complex bandaide, we've decided to "punt" on this issue of the time being.





