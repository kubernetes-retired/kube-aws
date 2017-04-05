package model

import (
	"strconv"
	"strings"
)

type CustomSystemdUnit struct {
	Name        string `yaml:"name"`
	Command     string `yaml:"command"`
	Content     string `yaml:"content"`
	Enable      bool   `yaml:"enable,omitempty"`
	Runtime     bool   `yaml:"runtime,omitempty"`
	UnknownKeys `yaml:",inline"`
}

func (c CustomSystemdUnit) ContentArray() []string {
	return strings.Split(c.Content, "\n")
}

func (c CustomSystemdUnit) EnableString() string {
	return strconv.FormatBool(c.Enable)
}

func (c CustomSystemdUnit) RuntimeString() string {
	return strconv.FormatBool(c.Runtime)
}
