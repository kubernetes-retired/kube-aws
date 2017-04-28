package model

type Autoscaling struct {
	ClusterAutoscaler ClusterAutoscaler `yaml:"clusterAutoscaler,omitempty"`
}
