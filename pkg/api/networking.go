package api

type Networking struct {
	AmazonVPC   AmazonVPC   `yaml:"amazonVPC,omitempty"`
	SelfHosting SelfHosting `yaml:"selfHosting,omitempty"`
}

type SelfHosting struct {
	Type            string           `yaml:"type"`
	Typha           bool             `yaml:"typha"`
	TyphaResources  ComputeResources `yaml:"typhaResources,omitempty"`
	CalicoNodeImage Image            `yaml:"calicoNodeImage"`
	CalicoCniImage  Image            `yaml:"calicoCniImage"`
	FlannelImage    Image            `yaml:"flannelImage"`
	FlannelCniImage Image            `yaml:"flannelCniImage"`
	TyphaImage      Image            `yaml:"typhaImage"`
	FlannelConfig   FlannelConfig    `yaml:"flannelConfig"`
	CalicoConfig    CalicoConfig     `yaml:"calicoConfig"`
}

type FlannelConfig struct {
	SubnetLen int32 `yaml:"subnetLen"`
}

type CalicoConfig struct {
	VxlanMode bool `yaml:"vxlanMode"`
}
