package cmd

import "github.com/coreos/kube-aws/cmd/nodepool"

func init() {
	RootCmd.AddCommand(nodepool.NodePoolCmd)
}
