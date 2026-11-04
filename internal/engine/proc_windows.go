//go:build windows

package engine

import (
	"os/exec"
	"syscall"
)

// applySysProcAttr keeps engine subprocesses invisible: a GUI must not flash
// a console window for every PowerShell call.
func applySysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
