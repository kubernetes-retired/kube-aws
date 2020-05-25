package cmd

import (
	"fmt"

	"github.com/kubernetes-incubator/kube-aws/core/root"
	"github.com/kubernetes-incubator/kube-aws/flatcar/amiregistry"
	"github.com/kubernetes-incubator/kube-aws/logger"
	"github.com/kubernetes-incubator/kube-aws/model"
	"github.com/spf13/cobra"
)

var (
	cmdAmi = &cobra.Command{
		Use:          "ami",
		Short:        "Compare AMIID of cluster.yaml VS the last release",
		Long:         ``,
		RunE:         runCmdAmi,
		SilenceUsage: true,
	}
)

func init() {
	RootCmd.AddCommand(cmdAmi)

}

func runCmdAmi(_ *cobra.Command, _ []string) error {
	opts := root.NewOptions(true, true)
	cluster, err := root.ClusterFromFile(configPath, opts, false)
	if err != nil {
		return fmt.Errorf("failed to read cluster config: %v", err)
	}

	region := cluster.ControlPlane().Region.Name
	channel := string(cluster.ControlPlane().ReleaseChannel)

	releaseChannel := model.ReleaseChannel(channel)
	amiID, err := amiregistry.GetAMI(region, releaseChannel)
	if err != nil {
		return fmt.Errorf("Impossible to retrieve FlatCar AMI for region %s, channel %s", region, channel)
	}

	if cluster.ControlPlane().AmiId == amiID {
		logger.Infof("AmiID up to date")
		return nil
	}

	successMsg := `
The Flatcar AmiId for region %s and release channel %s is different than the one in cluster definition.

Cluster.yaml:
- amiId: %s
+ amiId: %s
`
	logger.Infof(successMsg, region, channel, cluster.ControlPlane().AmiId, amiID)
	return nil
}
