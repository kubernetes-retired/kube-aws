package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/cluster"
)

var (
	cmdStatus = &cobra.Command{
		Use:   "status",
		Short: "Describe an existing Kubernetes cluster",
		Long:  ``,
		Run:   runCmdStatus,
	}
)

func init() {
	cmdRoot.AddCommand(cmdStatus)
}

func runCmdStatus(cmd *cobra.Command, args []string) {
	cfg := cluster.NewDefaultConfig(VERSION)
	err := cluster.DecodeConfigFromFile(cfg, rootOpts.ConfigPath)
	if err != nil {
		stderr("Unable to load cluster config: %v", err)
		os.Exit(1)
	}

	c := cluster.New(cfg, newAWSConfig(cfg))

	info, err := c.Info()
	if err != nil {
		stderr("Failed fetching cluster info: %v", err)
		os.Exit(1)
	}

	fmt.Print(info.String())
}
