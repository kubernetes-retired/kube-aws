package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/spf13/cobra"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/cluster"
)

var (
	// set by build script
	VERSION                    = "UNKNOWN"
	DefaultArtifactURLTemplate = "https://coreos-kubernetes.s3.amazonaws.com/%s"

	cmdRoot = &cobra.Command{
		Use:   "kube-aws",
		Short: "Manage Kubernetes clusters on AWS",
		Long:  ``,
	}

	rootOpts struct {
		AWSDebug   bool
		ConfigPath string
	}
)

func init() {
	cmdRoot.PersistentFlags().StringVar(&rootOpts.ConfigPath, "config", "cluster.yaml", "Location of kube-aws cluster config file")
	cmdRoot.PersistentFlags().BoolVar(&rootOpts.AWSDebug, "aws-debug", false, "Log debug information from aws-sdk-go library")
}

func main() {
	cmdRoot.Execute()
}

func stderr(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

func newAWSConfig(cfg *cluster.Config) *aws.Config {
	c := aws.NewConfig()
	c = c.WithRegion(cfg.Region)
	if rootOpts.AWSDebug {
		c = c.WithLogLevel(aws.LogDebug)
	}
	return c
}
