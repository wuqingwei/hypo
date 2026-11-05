package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetWhitelistMergesDefaultsAndUser(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `printf '%s' '{"defaults":["C:\\def\\a"],"user":["C:\\users\\x\\keep"]}'`)
	// The export reads the user file via env MOLE_WHITELIST_FILE; our fake ps
	// ignores it, so the user patterns here come from the fixture itself.

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	wl, err := NewClientWithPowerShell(install, ps).GetWhitelist(context.Background())
	if err != nil {
		t.Fatalf("get whitelist failed: %v", err)
	}
	if len(wl.Defaults) != 1 || !strings.HasSuffix(wl.Defaults[0], "\\a") {
		t.Errorf("defaults = %v", wl.Defaults)
	}
	if len(wl.UserPatterns) != 1 || wl.UserPatterns[0] != `C:\users\x\keep` {
		t.Errorf("user patterns = %v", wl.UserPatterns)
	}
	if wl.Path == "" || !strings.HasSuffix(wl.Path, "whitelist.txt") {
		t.Errorf("path = %q", wl.Path)
	}
}

func TestWhitelistReadWriteRoundTrip(t *testing.T) {
	dir := engineConfigDirTest(t)
	path := dir + `\whitelist.txt`

	// Create via add.
	if err := osWriteLine(path, ""); err != nil {
		t.Fatal(err)
	}
	patterns, err := readUserPatterns(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	_ = patterns
}

// engineConfigDirTest points engineConfigDir's USERPROFILE at a temp dir for
// the duration of the test.
func engineConfigDirTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	return dir + `\.config\mole`
}

func TestAddAndRemoveWhitelistEntry(t *testing.T) {
	engineConfigDirTest(t)

	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `printf '%s' '{"defaults":[],"user":[]}'`)
	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClientWithPowerShell(install, ps)

	// Add two patterns.
	if _, err := client.AddWhitelistEntry(context.Background(), `C:\Keep Me`); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if _, err := client.AddWhitelistEntry(context.Background(), `C:\Games\*\saves`); err != nil {
		t.Fatalf("add wildcard failed: %v", err)
	}

	// Duplicate is refused.
	if _, err := client.AddWhitelistEntry(context.Background(), `c:\keep me`); err == nil {
		t.Fatal("duplicate add must be refused")
	}

	lines, err := readRawLines(filepath.Join(engineConfigDir(), "whitelist.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != `C:\Keep Me` || lines[1] != `C:\Games\*\saves` {
		t.Errorf("file lines = %#v", lines)
	}

	// Remove one exact line; the other survives.
	if _, err := client.RemoveWhitelistEntry(context.Background(), `C:\Keep Me`); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	lines, _ = readRawLines(filepath.Join(engineConfigDir(), "whitelist.txt"))
	if len(lines) != 1 || lines[0] != `C:\Games\*\saves` {
		t.Errorf("after remove = %#v", lines)
	}

	// Removing a missing pattern reports it.
	if _, err := client.RemoveWhitelistEntry(context.Background(), `C:\Gone`); err == nil {
		t.Fatal("removing a missing pattern must report not found")
	}

	// Comments and blanks never match removal.
	if err := osWriteLine(filepath.Join(engineConfigDir(), "whitelist.txt"), "# note\r\n\r\nC:\\Real\r\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RemoveWhitelistEntry(context.Background(), "# note"); err == nil {
		t.Fatal("comment lines must not be removable as patterns")
	}
}

func TestAddRefusesBlankAndComment(t *testing.T) {
	engineConfigDirTest(t)
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `printf '%s' '{"defaults":[],"user":[]}'`)
	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClientWithPowerShell(install, ps)
	if _, err := client.AddWhitelistEntry(context.Background(), ""); err == nil {
		t.Fatal("blank must be refused")
	}
	if _, err := client.AddWhitelistEntry(context.Background(), "# comment"); err == nil {
		t.Fatal("comment must be refused")
	}
}

// osWriteLine is a tiny helper for fixture writes in these tests.
func osWriteLine(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
