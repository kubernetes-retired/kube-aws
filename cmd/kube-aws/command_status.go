package main

import (
	"fmt"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/cluster"
	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cmdStatus = &cobra.Command{
		Use:          "status",
		Short:        "Describe an existing Kubernetes cluster",
		Long:         ``,
		RunE:         runCmdStatus,
		SilenceUsage: true,
	}
)

func init() {
	cmdRoot.AddCommand(cmdStatus)
}

func runCmdStatus(cmd *cobra.Command, args []string) error {
	conf, err := config.ClusterFromFile(configPath)
	if err != nil {
		return fmt.Errorf("Failed to read cluster config: %v", err)
	}
	info, err := cluster.New(conf, false).Info()
	if err != nil {
		return fmt.Errorf("Failed fetching cluster info: %v", err)
	}

	fmt.Print(info.String())
	return nil
}
