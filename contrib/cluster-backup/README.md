# Restore
Restoring a Kubernetes environment (from a previous backup) may be executed through the use of the bash script: [kubernetes-on-aws-restore.sh](/Documentation/scripts/kubernetes-on-aws-backup.sh)    

The script was designed to be used by the cluster provisioner and assumes that their local machine has:
- 'kubectl' installed locally
- 'aws' installed locally
- 'jq' installed locally
- 'kubeconfig' configured correctly
- 'jq' configured correctly
- cluster is reachable 

The script will **(kubectl) create** Kubernetes resources per namespace. All namespaces (and associated resources) are restored by default with the exception of namespaces: 'kube-system' and 'default'.
To whitelist namespaces, include the desired namespace names as additional arguments after the initial BUCKET_URI argument.
The BUCKET_URI argument is the URL of the timestamped folder.

## Implementation
1) The script will initially validate that the namespaces it is to restore are not already existing. 
2) Kubernetes resources common to all namespaces (such as *StorageClass*) are created.
3) For each namespace to restore, the namespace is created and Kubernetes resources within the namespace are created.

### Considerations
- Restoring *PersistentVolumes* does not provision new volumes from AWS - it assumes such volumes already exist in AWS.
- You may have to tailor the script for your specific needs. 

 
### Example

Following the deletion of a cluster, another newly created cluster is to be restored with the same Kubernetes environment. 

A backup was previously exported to S3 to the path: 
```
s3://my-bucket-name/my-cluster-name/backup/17-05-04_13-48-33
```
After the new cluster has been provisioned and all nodes are up and operational, the provisioner restores the cluster:
```
$ kubernetes-on-aws-restore.sh s3://my-bucket-name/my-cluster-name/backup/17-05-04_13-48-33
```
