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
