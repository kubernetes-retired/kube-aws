# Kubernetes Dashboard Access and Authentication

## Default Setup

    kubernetesDashboard:
      adminPrivileges: true
      insecureLogin: false

In the default setup the Dashboard is configured using the `--auto-generate-certificates` flag and has Admin privileges

###### Access the dashboard  using `kubectl proxy` command:

    http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/


## Expose the Dashboard using a ELB with self-signed certificates

kubernetesDashboard:
  adminPrivileges: false
  insecureLogin: false

Ex.

    apiVersion: v1
    kind: Service
    metadata:
      annotations:
        external-dns.alpha.kubernetes.io/hostname: "kubedash.example.com"
      name: kubernetes-dashboard
      labels:
        run: kubernetes-dashboard
      namespace: kube-system
    spec:
      type: LoadBalancer
    # uncomment if you want to restrict the access to allowed IP's
    #  loadBalancerSourceRanges:
    #  - x.x.x.x/32
      ports:
      - port: 443
        targetPort: 8443
        protocol: TCP
      selector:
        k8s-app: kubernetes-dashboard   
###### Access the dashboard  using `kubectl proxy` command:

    http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/


## Expose the Dashboard using a ELB with trusted certificates

    kubernetesDashboard:
      adminPrivileges: false
      insecureLogin: true

Ex.

    apiVersion: v1
    kind: Service
    metadata:
      annotations:
        service.beta.kubernetes.io/aws-load-balancer-ssl-cert:
        #replace with your certificate ARN
        arn:aws:acm:us-east-1:XXXXXXXXXX:certificate/xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxx
        service.beta.kubernetes.io/aws-load-balancer-backend-protocol: http
        external-dns.alpha.kubernetes.io/hostname: "kubedash.example.com"
      name: kubernetes-dashboard
      labels:
        run: kubernetes-dashboard
      namespace: kube-system
    spec:
      type: LoadBalancer
      ports:
      - port: 443
        targetPort: 9090
        protocol: TCP
      selector:
        k8s-app: kubernetes-dashboard  

###### Access the dashboard  using `kubectl proxy` command:

    http://localhost:8001/api/v1/namespaces/kube-system/services/http:kubernetes-dashboard:/proxy/


## Expose the Dashboard using a Ingress with trusted certificates

    kubernetesDashboard:
      adminPrivileges: false
      insecureLogin: true

Ex.

    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: kubernetes-dashboard
      annotations:
        kubernetes.io/ingress.class: nginx
        kubernetes.io/tls-acme: "true"
        ingress.kubernetes.io/ssl-redirect: "true"
        ingress.kubernetes.io/use-port-in-redirects: "true"
      namespace: kube-system
    spec:
      tls:
      - hosts:
        - kubedash.example.com
        secretName: kubedash-tls
      rules:
      - host: kubedash.example.com
        http:
          paths:
          - path: /
            backend:
              serviceName: kubernetes-dashboard
              servicePort: 9090

###### Access the dashboard  using `kubectl proxy` command:

    http://localhost:8001/api/v1/namespaces/kube-system/services/http:kubernetes-dashboard:/proxy/

## Authentication using a token

Ex.

**Create a new ServiceAccount**

    kubectl create serviceaccount k8sadmin -n kube-system

**Create a ClusterRoleBinding with Cluster Admin Privileges**

    kubectl create clusterrolebinding k8sadmin --clusterrole=cluster-admin --serviceaccount=kube-system:k8sadmin

**Get the token**

    kubectl get secret -n kube-system | grep k8sadmin | cut -d " " -f1 | xargs -n 1 | xargs kubectl get secret  -o 'jsonpath={.data.token}' -n kube-system | base64 --decode
