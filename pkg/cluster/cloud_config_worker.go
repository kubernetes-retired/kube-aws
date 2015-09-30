package cluster

var baseWorkerCloudConfig = `#cloud-config
coreos:
  update:
    reboot-strategy: "off"

  flannel:
    interface: $private_ipv4
    etcd_endpoints: http://10.0.0.50:2379

  units:
  - name: install-worker.service
    command: start
    content: |
      [Service]
      ExecStart=/bin/bash /tmp/install-worker.sh
      Type=oneshot

write_files:
- path: /run/coreos-kubernetes/options.env
  content: |
    ETCD_ENDPOINTS=http://10.0.0.50:2379
    CONTROLLER_ENDPOINT=https://10.0.0.50
    ARTIFACT_URL={{ ArtifactURL }}

- path: /tmp/install-worker.sh
  content: |
    #!/bin/bash

    exec bash -c "$(curl --fail --silent --show-error --location '{{ ArtifactURL }}/scripts/install-worker.sh')"

- path: /etc/kubernetes/ssl/ca.pem
  encoding: base64
  content: {{ CACert|base64 }}

- path: /etc/kubernetes/ssl/worker.pem
  encoding: base64
  content: {{ WorkerCert|base64 }}

- path: /etc/kubernetes/ssl/worker-key.pem
  encoding: base64
  content: {{ WorkerKey|base64 }}
`
