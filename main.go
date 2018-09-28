package main

import (
	"os"

	"fmt"
	"github.com/kubernetes-incubator/kube-aws/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		switch e := err.(type) {
		case *cmd.ExitError:
			fmt.Fprintf(os.Stderr, "%s\n", e.Error())
			os.Exit(e.Code)
		}
		os.Exit(1)
	}
}
