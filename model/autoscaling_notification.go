package model

import (
	"errors"
)

type AutoscalingNotification struct {
	IAMConfig `yaml:"iam,omitempty"`
}

func (n AutoscalingNotification) RoleLogicalName() (string, error) {
	if !n.RoleManaged() {
		return "", errors.New("[BUG] RoleLogicalName should not be called when an existing autoscaling notification role was specified")
	}
	return "ASGNotificationRole", nil
}

func (n AutoscalingNotification) RoleArn() (string, error) {
	return n.IAMConfig.Role.ARN.OrGetAttArn(func() (string, error) { return n.RoleLogicalName() })
}

func (n AutoscalingNotification) RoleManaged() bool {
	return !n.IAMConfig.Role.HasArn()
}
