//go:build !windows

package engine

import "os/exec"

// applySysProcAttr is a no-op off Windows; tests run here with fake
// executables.
func applySysProcAttr(cmd *exec.Cmd) {}
