package model

import "errors"

const (
	stable = "stable"
	beta   = "beta"
	alpha  = "alpha"
	edge   = "edge"
)

type ReleaseChannel string

func (ch ReleaseChannel) IsValid() error {
	switch ch {
	case stable, beta, alpha, edge:
		return nil
	}
	return errors.New("Invalid Release Channel")
}

func DefaultReleaseChannel() ReleaseChannel {
	return ReleaseChannel(stable)
}
