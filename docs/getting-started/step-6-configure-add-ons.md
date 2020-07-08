# Configuring Kubernetes Add-ons

kube-aws has built-in supports for several Kubernetes add-ons known to require additional configurations beforehand.

## cluster-autoscaler

[cluster-autoscaler](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler) is an add-on which automatically
scales in/out your k8s cluster by removing/adding worker nodes according to resource utilization per node.

To enable cluster-autoscaler, add the below settings to your cluster.yaml:

```yaml
addons:
  clusterAutoscaler:
    enabled: true
worker:
  nodePools:
  - name: scaled
    autoScalingGroup:
      minSize: 1
      maxSize: 10
    autoscaling:
      clusterAutoscaler:
        enabled: true
  - name: notScaled
    autoScalingGroup:
      minSize: 2
      maxSize: 4
```

The above example configuration would:

* By `addons.clusterAutoscaler.enabled`:
  * Provide controller nodes appropriate IAM permissions to call necessary AWS APIs from CA
  * Create a k8s deployment to run CA on one of controller nodes, so that CA can utilize the IAM permissions
* By `worker.nodePools[0].autoscaling.clusterAutoscaler.enabled`:
  * If there are unschedulable, pending pod(s) that is requesting more capacity, CA will add more nodes to the `scaled` node pool, up until the max size `10`
  * If there are no unschdulable, pending pod(s) that is waiting for more capacity and one or more nodes are in low utlization, CA will remove node(s), down until the min size `1`
* The second node pool `notScaled` is scaled manually by YOU, because you had not the autoscaling on it(=missing `autoscaling.clusterAutoscaler.enabled`)

## kube2iam / kiam
 
[kube2iam](https://github.com/jtblin/kube2iam) and [kiam](https://github.com/uswitch/kiam) are add-ons which provides IAM credentials for target IAM roles to pods running inside a Kubernetes cluster based on annotations.
To allow kube2iam or kiam deployed to worker and controller nodes to assume target roles, you need the following configurations.

1. IAM roles associated to worker and controller nodes requires an IAM policy:
 
  ```json
  {
    "Action": "sts:AssumeRole",
    "Resource": "*",
    "Effect": "Allow"
  }
  ```

  To add the policy to controller nodes, set `kubeAwsPlugins.kube2iam.enabled` or `kubeAwsPlugins.kiam.enabled` to `true` in your `cluster.yaml` (but not both).

2. Target IAM roles needs to change trust relationships to allow kube-aws worker/controller IAM role to assume the target roles.

  As CloudFormation generates unpredictable role names containing random IDs by default, it is recommended to make them predictable at first so that you can easily automate configuring trust relationships afterwards.
  To make worker/controller role names predictable, set `controller.iam.role.name` for controller and `worker.nodePools[].iam.role.name` for worker nodes.
  `iam.role.name`s becomes suffixes of the resulting worker/controller role names. 
  
  Please beware that configuration of target roles' trust relationships are out-of-scope of kube-aws.
  Please see [the part of kube2iam doc](https://github.com/jtblin/kube2iam#iam-roles) or [the part of the kiam doc](https://github.com/uswitch/kiam/blob/master/docs/IAM.md)for more information.
  Basically, you need to point `Principal` to the ARN of a resulting worker/controller IAM role which would look like `arn:aws:iam::<your aws account id>:role/<stack-name>-<managed iam role name>`. 

Finally, an example `cluster.yaml` usable with kube2iam would look like:

```yaml
# for controller nodes
controller:
  iam:
    role:
      name: mycontrollerrole
 
kubeAwsPlugins:
  kube2iam:
    enabled: true

# for worker nodes
worker:
  nodePools:
  - name: mypool
    iam:
      role:
        name: myworkerrole
 ```

See the relevant GitHub issues for [kube2iam](https://github.com/kubernetes-incubator/kube-aws/issues/253) and [kiam](https://github.com/kubernetes-incubator/kube-aws/issues/1055) or more information.

You can reference controller and worker IAM Roles in a separate CloudFormation stack that provides roles to assume:

```yaml
...
Parameters:
  KubeAWSStackName:
    Type: String
Resources:
  IAMRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Statement:
          - Effect: Allow
            Action: sts:AssumeRole
            Principal:
              Service: ec2.amazonaws.com
          - Effect: Allow
            Action: sts:AssumeRole
            Principal:
              AWS:
                Fn::ImportValue: !Sub "${KubeAWSStackName}-ControllerIAMRoleArn"
          - Effect: Allow
            Action: sts:AssumeRole
            Principal:
              AWS:
                Fn::ImportValue: !Sub "${KubeAWSStackName}-NodePool<Node Pool Name>WorkerIAMRoleArn"
      ...
```

## aws-iam-authenticator

[aws-iam-authenticator](https://github.com/kubernetes-sigs/aws-iam-authenticator) is an add-on which permits users, roles, nodes, etc. to access the cluster using IAM credentials. This plugin cannot be enabled when a cluster is initially created. To successfully enable it, you must update an existing stack.

### Enable aws-iam-authenticator

1. Merge these settings into your cluster.yaml

```yaml
controller:
  iam:
    role:
      name: controller

kubeAwsPlugins:
  awsIamAuthenticator:
    enabled: true
```

By controlling the name of the role associated with the controller you will find it easier to grant the necessary permissions. The name of the role will become CLUSTER_NAME-REGION-IAM_ROLE_NAME.

**NOTE** Applying this update will delete the existing role for the controller, replacing it with a new one.

1. The IAM role associated with controller nodes requires an IAM policy:
 
  ```json
  {
    "Action": "sts:AssumeRole",
    "Resource": "*",
    "Effect": "Allow"
  }
  ```

1. `kube-aws render credentials`
    * The resulting cert and key will be signed by the default CA
1. `kube-aws render stack`
1. Consult the [aws-iam-aithentication docs](https://github.com/kubernetes-sigs/aws-iam-authenticator/#4-create-iam-roleuser-to-kubernetes-usergroup-mappings), to configure roles and the config map.
    * the config map is in `plugins/aws-iam-authenticator/manifest/aws-auth-cm.yaml`
    * server command line arguments are in `plugins/aws-iam-authenticator/manifest/daemonset.yaml`
    * don't forget to update kubeconfig
1. The aws-iam-authenticator docker image is only available from ECR. If you are not regularly authenticating to ECR in us-west-2, you will probably need to copy the image to your own docker registry. 
    1. Login to ECR
        * `aws --region us-west-2 ecr get-login-password | docker login --username AWS --password-stdin 602401143452.dkr.ecr.us-west-2.amazonaws.com`
    1. Choose your preferred image to [pull](https://github.com/kubernetes-sigs/aws-iam-authenticator/releases), ie
        * `docker pull 602401143452.dkr.ecr.us-west-2.amazonaws.com/amazon/aws-iam-authenticator:v0.5.1-alpine-3.7`
    1. Re-tag the image
        * `docker tag 602401143452.dkr.ecr.us-west-2.amazonaws.com/amazon/aws-iam-authenticator:v0.5.1-alpine-3.7 YOUR_ORG/aws-iam-authenticator:v0.5.1-alpine-3.7`
    1. Push the new tag
        * `docker push YOUR_ORG/aws-iam-authenticator:v0.5.1-alpine-3.7`
    1. Update the `image` key in `plugins/aws-iam-authenticator/manifest/daemonset.yaml`
1. `kube-aws apply`

### Delete an aws-iam-authenticator enabled cluster

To successfully delete a cluster with aws-iam-authenticator enabled you must
work around a quirk in cloud formation. You have two options.

#### Delete the control plane stack manually and then delete the cluster

1.  https://console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks?filteringText=&filteringStatus=active&viewNested=true&hideStacks=false
    1. Navigate to the control plane stack of the cluster you'd like to delete
    1. Click the delete button
        1. Choose to delete the sub stack
        1. Type in the confirmation
1. Follow the rest of the instructions to [destroy your cluster][getting-started-step-7] 

#### Disable aws-iam-authenticator and then delete the cluster

1. Set `kubeAwsPlugins.awsIamAuthenticator.enabled` to `false`
1. `kube-aws apply`
1. Follow the rest of the instructions to [destroy your cluster][getting-started-step-7] 

When you are done with your cluster, [destroy your cluster][getting-started-step-7]

[getting-started-step-1]: step-1-configure.md
[getting-started-step-2]: step-2-render.md
[getting-started-step-3]: step-3-launch.md
[getting-started-step-4]: step-4-update.md
[getting-started-step-5]: step-5-add-node-pool.md
[getting-started-step-6]: step-6-configure-add-ons.md
[getting-started-step-7]: step-7-destroy.md
