package main

import (
	"os"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cmdRoot = &cobra.Command{
		Use:   "kube-aws",
		Short: "Manage Kubernetes clusters on AWS",
		Long:  ``,
	}
)

const configPath = "cluster.yaml"

var stackTemplateOptions = config.StackTemplateOptions{
	TLSAssetsDir:          "credentials",
	ControllerTmplFile:    "userdata/cloud-config-controller",
	WorkerTmplFile:        "userdata/cloud-config-worker",
	StackTemplateTmplFile: "stack-template.json",
}

func main() {
	if err := cmdRoot.Execute(); err != nil {
		os.Exit(2)
	}
}
