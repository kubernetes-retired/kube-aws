package model

import (
	"fmt"
	"regexp"
)

type IAMConfig struct {
	Role            IAMRole            `yaml:"role,omitempty"`
	InstanceProfile IAMInstanceProfile `yaml:"instanceProfile,omitempty"`
	UnknownKeys     `yaml:",inline"`
}

type IAMRole struct {
	Name            string             `yaml:"name,omitempty"`
	ManagedPolicies []IAMManagedPolicy `yaml:"managedPolicies,omitempty"`
}

type IAMManagedPolicy struct {
	Arn string `yaml:"arn,omitempty"`
}

type IAMInstanceProfile struct {
	Arn string `yaml:"arn,omitempty"`
}

func (c IAMConfig) Validate() error {

	managedPolicyRegexp := regexp.MustCompile(`arn:aws:iam::((\d{12})|aws):policy/([a-zA-Z0-9-=,\\.@_]{1,128})`)
	instanceProfileRegexp := regexp.MustCompile(`arn:aws:iam::(\d{12}):instance-profile/([a-zA-Z0-9-=,\\.@_]{1,128})`)
	for _, policy := range c.Role.ManagedPolicies {
		if !managedPolicyRegexp.MatchString(policy.Arn) {
			return fmt.Errorf("invalid managed policy arn, your managed policy must match this (=arn:aws:iam::(YOURACCOUNTID|aws):policy/POLICYNAME), provided this (%s)", policy.Arn)
		}
	}
	if c.InstanceProfile.Arn != "" {
		if !instanceProfileRegexp.MatchString(c.InstanceProfile.Arn) {
			return fmt.Errorf("invalid instance profile, your instance profile must match (=arn:aws:iam::YOURACCOUNTID:instance-profile/INSTANCEPROFILENAME), provided (%s)", c.InstanceProfile.Arn)
		}

	}
	return nil

}
