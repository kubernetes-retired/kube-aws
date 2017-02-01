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

func (i Identifier) Ref(logicalNameProvider func() string) string {
	if i.IDFromStackOutput != "" {
		return fmt.Sprintf(`{ "Fn::ImportValue" : %q }`, i.IDFromStackOutput)
	} else if i.ID != "" {
		return fmt.Sprintf(`"%s"`, i.ID)
	} else {
		return fmt.Sprintf(`{ "Ref" : %q }`, logicalNameProvider())
	}
}

// RefOrError should be used instead of Ref where possible so that kube-aws can print a more useful error message with
// the line number for the stack-template.json when there's an error.
func (i Identifier) RefOrError(logicalNameProvider func() (string, error)) (string, error) {
	if i.IDFromStackOutput != "" {
		return fmt.Sprintf(`{ "Fn::ImportValue" : %q }`, i.IDFromStackOutput), nil
	} else if i.ID != "" {
		return fmt.Sprintf(`"%s"`, i.ID), nil
	} else {
		logicalName, err := logicalNameProvider()
		if err != nil {
			return "", fmt.Errorf("failed to get id or ref: %v", err)
		}
		return fmt.Sprintf(`{ "Ref" : %q }`, logicalName), nil
	}
}
