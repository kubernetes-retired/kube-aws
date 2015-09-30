package main

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/cluster"
)

var (
	cmdDestroy = &cobra.Command{
		Use:   "destroy",
		Short: "Destroy an existing Kubernetes cluster",
		Long:  ``,
		Run:   runCmdDestroy,
	}
)

func init() {
	cmdRoot.AddCommand(cmdDestroy)
}

func runCmdDestroy(cmd *cobra.Command, args []string) {
	cfg := cluster.NewDefaultConfig(VERSION)
	err := cluster.DecodeConfigFromFile(cfg, rootOpts.ConfigPath)
	if err != nil {
		stderr("Unable to load cluster config: %v", err)
		os.Exit(1)
	}

	c := cluster.New(cfg, newAWSConfig(cfg))

	if err := c.Destroy(); err != nil {
		stderr("Failed destroying cluster: %v", err)
		os.Exit(1)
	}

	clusterDir := path.Join("clusters", cfg.ClusterName)
	if err := os.RemoveAll(clusterDir); err != nil {
		stderr("Failed removing local cluster directory: %v", err)
		os.Exit(1)
	}

	fmt.Println("Destroyed cluster")
}
