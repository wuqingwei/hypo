// Package engine locates and drives the Mole CLI engine.
//
// Hypo never implements cleanup itself: every delete, uninstall and recycle
// goes through the engine's own guarded pipeline (Remove-SafeItem and friends,
// see docs/ARCHITECTURE.md). This package only finds the engine tree, runs
// its commands and decodes their JSON output.
package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvVar overrides every search location. It exists for development and for
// users whose engine lives outside the standard places.
const EnvVar = "HYPO_ENGINE_ROOT"

// Source says where an Installation was found.
type Source string

const (
	// SourceEnv is an explicit HYPO_ENGINE_ROOT.
	SourceEnv Source = "env"
	// SourcePackaged is the engine folder shipped beside the Hypo executable.
	SourcePackaged Source = "packaged"
	// SourceSibling is a Mole checkout beside the Hypo source tree.
	SourceSibling Source = "sibling"
)

// ErrNotFound is returned when no engine tree exists in any search location.
var ErrNotFound = errors.New("mole engine not found")

// Installation is a validated Mole CLI engine tree.
type Installation struct {
	// Root is the engine root directory (contains mole.ps1).
	Root string
	// MoleScript is the absolute path to mole.ps1.
	MoleScript string
	// Version is the trimmed VERSION file contents; "" when unreadable.
	Version string
	// Source records how the tree was found.
	Source Source

	// StatusBin and AnalyzeBin are the native Go tools when already present,
	// following the engine's own resolution (bin/<name>.exe, then
	// <name>.exe at the root). Empty means: call mole.ps1 instead, which
	// downloads or builds the tool on demand.
	StatusBin  string
	AnalyzeBin string
}

// Locate finds the engine. Search order:
//
//  1. HYPO_ENGINE_ROOT (an invalid value is an error, not silently skipped)
//  2. <exe dir>/engine (how installers ship the engine beside Hypo)
//  3. a Mole checkout beside the Hypo source tree, for development
func Locate() (*Installation, error) {
	if root := strings.TrimSpace(os.Getenv(EnvVar)); root != "" {
		install, err := installationAt(root, SourceEnv)
		if err != nil {
			return nil, fmt.Errorf("%s=%q is not a usable engine tree: %w", EnvVar, root, err)
		}
		return install, nil
	}
	return locateFrom(executableDir())
}

// locateFrom is Locate with an explicit starting directory, so tests can
// construct candidate layouts in temp folders.
func locateFrom(exeDir string) (*Installation, error) {
	if install, err := installationAt(filepath.Join(exeDir, "engine"), SourcePackaged); err == nil {
		return install, nil
	}
	// Development layout: .../opensource/Hypo and .../opensource/Mole. Walk a
	// few ancestors of the executable so it works from the repo root and from
	// build output folders.
	for dir, depth := exeDir, 0; depth < 4 && dir != filepath.Dir(dir); dir, depth = filepath.Dir(dir), depth+1 {
		for _, name := range []string{"Mole", "mole"} {
			if install, err := installationAt(filepath.Join(dir, name), SourceSibling); err == nil {
				return install, nil
			}
		}
	}
	return nil, ErrNotFound
}

// installationAt validates one candidate root and fills in the tool paths.
// It mirrors lib/core/tui_binaries.ps1's resolution order.
func installationAt(root string, src Source) (*Installation, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	script := filepath.Join(abs, "mole.ps1")
	if !isRegularFile(script) {
		return nil, fmt.Errorf("mole.ps1 not found in %s", abs)
	}
	// base.ps1 is the safety core; a tree without it is not a Mole engine.
	if !isRegularFile(filepath.Join(abs, "lib", "core", "base.ps1")) {
		return nil, fmt.Errorf("lib/core/base.ps1 not found in %s: not a Mole engine tree", abs)
	}
	install := &Installation{Root: abs, MoleScript: script, Source: src}
	if data, err := os.ReadFile(filepath.Join(abs, "VERSION")); err == nil {
		install.Version = strings.TrimSpace(string(data))
	}
	install.StatusBin = resolveTool(abs, "status")
	install.AnalyzeBin = resolveTool(abs, "analyze")
	return install, nil
}

// resolveTool follows the engine's own lookup: bin/<name>.exe first, then
// <name>.exe at the root.
func resolveTool(root, name string) string {
	for _, rel := range []string{filepath.Join("bin", name+".exe"), name + ".exe"} {
		if p := filepath.Join(root, rel); isRegularFile(p) {
			return p
		}
	}
	return ""
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
