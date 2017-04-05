package model

import (
	"fmt"
	"strings"
)

type CustomFile struct {
	Path        string `yaml:"path"`
	Permissions uint   `yaml:"permissions"`
	Content     string `yaml:"content"`
	UnknownKeys `yaml:",inline"`
}

func (c CustomFile) ContentArray() []string {
	// CustomFiles needs an array so that we can inline it.
	return strings.Split(c.Content, "\n")
}

func (c CustomFile) PermissionsString() string {
	// We also need to write out octal notation for permissions.
	return fmt.Sprintf("0%o", c.Permissions)
}
