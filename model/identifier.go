package model

import (
	"fmt"
)

type Identifier struct {
	ID                string `yaml:"id,omitempty"`
	IDFromStackOutput string `yaml:"idFromStackOutput,omitempty"`
}

func (i Identifier) HasIdentifier() bool {
	return i.ID != "" || i.IDFromStackOutput != ""
}

func (i Identifier) Ref(logicalName string) string {
	if i.IDFromStackOutput != "" {
		return fmt.Sprintf(`{ "ImportValue" : %q }`, i.IDFromStackOutput)
	} else if i.ID != "" {
		return fmt.Sprintf(`"%s"`, i.ID)
	} else {
		return fmt.Sprintf(`{ "Ref" : %q }`, logicalName)
	}
}

func (i Identifier) IdOrRef(refProvider func() (string, error)) (string, error) {
	if i.IDFromStackOutput != "" {
		return fmt.Sprintf(`{ "ImportValue" : %q }`, i.IDFromStackOutput), nil
	} else if i.ID != "" {
		return fmt.Sprintf(`"%s"`, i.ID), nil
	} else {
		logicalName, err := refProvider()
		if err != nil {
			return "", fmt.Errorf("failed to get id or ref: %v", err)
		}
		return fmt.Sprintf(`{ "Ref" : %q }`, logicalName), nil
	}
}
