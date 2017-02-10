package cmd

import (
	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:   "kube-aws",
		Short: "Manage Kubernetes clusters on AWS",
		Long:  ``,
	}

	configPath = "cluster.yaml"
)
