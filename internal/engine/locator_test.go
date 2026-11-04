package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeEngineTree creates a directory shaped like a Mole engine tree.
func writeEngineTree(t *testing.T, root string, withStatusTool bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "lib", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"mole.ps1":          "# fake engine entry",
		"lib/core/base.ps1": "# fake safety core",
		"VERSION":           "V9.9.9-windows\n",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if withStatusTool {
		bin := filepath.Join(root, "bin")
		if err := os.MkdirAll(bin, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bin, "status.exe"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLocateFromPackagedEngineBesideExecutable(t *testing.T) {
	exeDir := t.TempDir()
	writeEngineTree(t, filepath.Join(exeDir, "engine"), true)

	install, err := locateFrom(exeDir)
	if err != nil {
		t.Fatalf("expected the packaged engine to be found: %v", err)
	}
	if install.Source != SourcePackaged {
		t.Errorf("source = %q, want %q", install.Source, SourcePackaged)
	}
	if install.Version != "V9.9.9-windows" {
		t.Errorf("version = %q, want V9.9.9-windows", install.Version)
	}
	if install.StatusBin == "" {
		t.Error("status tool under bin/ was not resolved")
	}
	if install.MoleScript != filepath.Join(install.Root, "mole.ps1") {
		t.Errorf("mole script path = %q", install.MoleScript)
	}
}

func TestLocateFromSiblingCheckoutThroughAncestors(t *testing.T) {
	base := t.TempDir()
	// .../base/build/bin/hypo.exe and .../base/Mole
	exeDir := filepath.Join(base, "build", "bin")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeEngineTree(t, filepath.Join(base, "Mole"), false)

	install, err := locateFrom(exeDir)
	if err != nil {
		t.Fatalf("expected the sibling checkout to be found: %v", err)
	}
	if install.Source != SourceSibling {
		t.Errorf("source = %q, want %q", install.Source, SourceSibling)
	}
	if install.StatusBin != "" {
		t.Errorf("status tool = %q, want empty when the tree has no binaries", install.StatusBin)
	}
}

func TestLocateFromRejectsTreeWithoutSafetyCore(t *testing.T) {
	base := t.TempDir()
	exeDir := filepath.Join(base, "hypo")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A folder with mole.ps1 but no lib/core/base.ps1 is not an engine.
	fake := filepath.Join(base, "Mole")
	if err := os.MkdirAll(fake, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fake, "mole.ps1"), []byte("# not mole"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := locateFrom(exeDir); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestLocateHonoursEnvVarAndFailsLoudlyOnGarbage(t *testing.T) {
	t.Setenv(EnvVar, "")

	root := t.TempDir()
	writeEngineTree(t, root, false)
	t.Setenv(EnvVar, root)
	install, err := Locate()
	if err != nil {
		t.Fatalf("env engine not found: %v", err)
	}
	if install.Source != SourceEnv || install.Root != root {
		t.Errorf("install = %+v", install)
	}

	t.Setenv(EnvVar, filepath.Join(t.TempDir(), "missing"))
	if _, err := Locate(); err == nil {
		t.Fatal("an invalid HYPO_ENGINE_ROOT must be an error, not a silent skip")
	}
}
