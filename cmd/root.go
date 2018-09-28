package cmd

import (
	"github.com/kubernetes-incubator/kube-aws/logger"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:   "kube-aws",
		Short: "Manage Kubernetes clusters on AWS",
		Long:  ``,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			colorEnabled, err := cmd.Flags().GetBool("color")
			if err != nil {
				panic(err)
			}
			ansi.DisableColors(!colorEnabled)
		},
	}

	configPath = "cluster.yaml"
)

func init() {
	RootCmd.SetOutput(logger.Writer(logger.StdErrOutput))
	RootCmd.PersistentFlags().BoolVarP(
		&logger.Silent,
		"silent",
		"s",
		false,
		"do not show messages",
	)
	RootCmd.PersistentFlags().BoolVarP(
		&logger.Verbose,
		"verbose",
		"v",
		false,
		"show debug messages",
	)
	RootCmd.PersistentFlags().BoolVarP(
		&logger.Color,
		"color",
		"",
		false,
		"use color for messages",
	)
}
