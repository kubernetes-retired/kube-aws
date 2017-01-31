package model

type InternetGateway struct {
	Identifier `yaml:",inline"`
}

func (g InternetGateway) ManageInternetGateway() bool {
	return !g.HasIdentifier()
}
