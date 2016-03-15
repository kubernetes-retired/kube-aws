package vmcheck

import (
	"github.com/coreos/coreos-cloudinit/Godeps/_workspace/src/github.com/sigma/bdoor"
)

// IsVirtualWorld returns whether the code is running in a VMware virtual machine or no
func IsVirtualWorld() bool {
	return bdoor.HypervisorPortCheck()
}
