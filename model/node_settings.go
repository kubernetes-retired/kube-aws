package model

type NodeSettings struct {
	NodeLabels NodeLabels `yaml:"nodeLabels"`
	Taints     Taints     `yaml:"taints"`
}

func (s NodeSettings) Validate() error {
	if err := s.Taints.Valid(); err != nil {
		return err
	}
	return nil
}
