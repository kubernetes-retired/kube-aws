package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/coreos/coreos-kubernetes/multi-node/aws/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cmdInit = &cobra.Command{
		Use:          "init",
		Short:        "Initialize default kube-aws cluster configuration",
		Long:         ``,
		RunE:         runCmdInit,
		SilenceUsage: true,
	}

	initOpts = config.Config{}
)

func init() {
	cmdRoot.AddCommand(cmdInit)
	cmdInit.Flags().StringVar(&initOpts.ClusterName, "cluster-name", "", "The name of this cluster. This will be the name of the cloudformation stack")
	cmdInit.Flags().StringVar(&initOpts.ExternalDNSName, "external-dns-name", "", "The hostname that will route to the api server")
	cmdInit.Flags().StringVar(&initOpts.Region, "region", "", "The AWS region to deploy to")
	cmdInit.Flags().StringVar(&initOpts.AvailabilityZone, "availability-zone", "", "The AWS availability-zone to deploy to")
	cmdInit.Flags().StringVar(&initOpts.KeyName, "key-name", "", "The AWS key-pair for ssh access to nodes")
	cmdInit.Flags().StringVar(&initOpts.KMSKeyARN, "kms-key-arn", "", "The ARN of the AWS KMS key for encrypting TLS assets")
}

func runCmdInit(cmd *cobra.Command, args []string) error {
	// Validate flags.
	required := []struct {
		name, val string
	}{
		{"--cluster-name", initOpts.ClusterName},
		{"--external-dns-name", initOpts.ExternalDNSName},
		{"--region", initOpts.Region},
		{"--availability-zone", initOpts.AvailabilityZone},
		{"--key-name", initOpts.KeyName},
		{"--kms-key-arn", initOpts.KMSKeyARN},
	}
	var missing []string
	for _, req := range required {
		if req.val == "" {
			missing = append(missing, strconv.Quote(req.name))
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("Missing required flag(s): %s", strings.Join(missing, ", "))
	}

	// Render the default cluster config.
	cfgTemplate, err := template.New("cluster.yaml").Parse(string(config.DefaultClusterConfig))
	if err != nil {
		return fmt.Errorf("Error parsing default config template: %v", err)
	}

	out, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("Error opening %s : %v", configPath, err)
	}
	defer out.Close()
	if err := cfgTemplate.Execute(out, initOpts); err != nil {
		return fmt.Errorf("Error exec-ing default config template: %v", err)
	}

	successMsg :=
		`Success! Created %s

Next steps:
1. (Optional) Edit %s to parameterize the cluster.
2. Use the "kube-aws render" command to render the stack template.
`

	fmt.Printf(successMsg, configPath, configPath)
	return nil
}
