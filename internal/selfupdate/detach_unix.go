//go:build !windows

package selfupdate

import (
	"os/exec"
	"syscall"
)

// detach puts the background updater in its own session so it fully outlives the
// short-lived parent CLI and is not tied to its controlling terminal.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
