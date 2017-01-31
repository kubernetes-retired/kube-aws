package model

type InternetGateway struct {
	Identifier    `yaml:",inline"`
	Preconfigured bool `yaml:"preconfigured,omitempty"`
}

func (g InternetGateway) ManageInternetGateway() bool {
	return !g.HasIdentifier()
}
