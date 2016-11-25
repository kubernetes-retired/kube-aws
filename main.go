package main

import (
	"github.com/coreos/kube-aws/cmd"
	"os"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}
