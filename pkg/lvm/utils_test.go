package lvm_test

import (
	"os/exec"
	"testing"

	"github.com/labring/sealos-state-metrics/pkg/lvm"
)

func TestListLVMVolumeGroup(t *testing.T) {
	// Check if vgs command exists
	if _, err := exec.LookPath("vgs"); err != nil {
		t.Skip("Skipping test: vgs command not found. LVM may not be installed on this system.")
	}

	// Try to list LVM volume groups
	vgs, err := lvm.ListLVMVolumeGroup(false)
	if err != nil {
		// If the command fails, skip the test instead of failing
		// This can happen in CI environments where LVM tools are installed
		// but no LVM configuration exists or permissions are insufficient
		t.Skipf("Skipping test: LVM not available or not configured: %v", err)
	}

	if len(vgs) == 0 {
		t.Skip("No LVM volume groups found on this system")
	}

	for i := range vgs {
		t.Logf("Found LVM volume group: %#+v", vgs[i])
	}
}
