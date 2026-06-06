//go:build windows

package selfupdate

import (
	"os/exec"
	"syscall"
)

// Windows process-creation flags: run the updater without a console and in its
// own process group so it survives the parent CLI exiting.
const (
	detachedProcess       = 0x00000008 // DETACHED_PROCESS
	createNewProcessGroup = 0x00000200 // CREATE_NEW_PROCESS_GROUP
)

// detach starts the background updater as a detached, console-less process so it
// fully outlives the short-lived parent CLI.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | createNewProcessGroup,
	}
}
