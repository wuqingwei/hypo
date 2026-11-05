package engine

import (
	"context"
	"strings"
	"testing"
)

const uninstallListFixture = `[
  {
    "name": "WinMerge 2.16.44",
    "id": "registry:HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\WinMerge_is1",
    "source": "Registry",
    "version": "2.16.44",
    "publisher": "WinMerge Team",
    "path": "C:\\Program Files\\WinMerge",
    "size": "12.4 MB",
    "size_bytes": 13002343
  },
  {
    "name": "\u5e94\u7528\u4e2d\u6587\u540d",
    "id": "package:App_1.0_x64__abc",
    "source": "WindowsStore",
    "version": null,
    "publisher": "Vendor",
    "path": null,
    "size": "Unknown",
    "size_bytes": null
  }
]`

func TestUninstallListDecodesInventory(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `printf '%s' '`+uninstallListFixture+`'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	apps, err := NewClientWithPowerShell(install, ps).UninstallList(context.Background())
	if err != nil {
		t.Fatalf("uninstall list failed: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("apps = %d, want 2", len(apps))
	}
	if apps[0].ID == "" || !strings.Contains(apps[0].ID, "WinMerge_is1") {
		t.Errorf("registry id did not round-trip: %q", apps[0].ID)
	}
	if apps[0].SizeBytes == nil || *apps[0].SizeBytes != 13002343 {
		t.Errorf("size bytes = %v", apps[0].SizeBytes)
	}
	// A null JSON value must stay null, not the zero string.
	if apps[1].Path != nil {
		t.Errorf("null path became %q", *apps[1].Path)
	}
	if apps[1].Name != "应用中文名" {
		t.Errorf("escaped CJK name did not decode: %q", apps[1].Name)
	}
}

func TestUninstallSelectPassesKeysAndCollectsNotices(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `
case " $* " in
  *" uninstall --select "*--yes*--json*) ;;
  *) echo "unexpected args: $*" >&2; exit 8 ;;
esac
printf 'NOTICE: The charge limit stays after uninstall.\n' >&2
printf '%s' '{"results":[{"id":"registry:X","name":"X","outcome":"uninstalled","detail":"exit code 0"},{"id":"registry:Y","name":"Y","outcome":"skipped","detail":"still installed"}],"incomplete":true}'
`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	run, err := NewClientWithPowerShell(install, ps).UninstallSelect(context.Background(), []string{"registry:X", "registry:Y"})
	if err != nil {
		t.Fatalf("uninstall select failed: %v", err)
	}
	if len(run.Result.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(run.Result.Results))
	}
	if run.Result.Results[1].Outcome != "skipped" {
		t.Errorf("second outcome = %q", run.Result.Results[1].Outcome)
	}
	if !run.Result.Incomplete {
		t.Error("incomplete flag lost")
	}
	if len(run.Notices) != 1 || !strings.Contains(run.Notices[0], "charge limit") {
		t.Errorf("notices = %v", run.Notices)
	}
}

func TestUninstallSelectRefusesEmptySelection(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `echo "must not run" >&2; exit 8`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewClientWithPowerShell(install, ps).UninstallSelect(context.Background(), nil); err == nil {
		t.Fatal("an empty selection must be refused before any process starts")
	}
}

func TestUninstallSelectReportsNotFound(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `printf '%s' '{"results":[],"incomplete":true,"not_found":["registry:Ghost"]}'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	run, err := NewClientWithPowerShell(install, ps).UninstallSelect(context.Background(), []string{"registry:Ghost"})
	if err != nil {
		t.Fatalf("uninstall select failed: %v", err)
	}
	if len(run.Result.NotFound) != 1 || run.Result.NotFound[0] != "registry:Ghost" {
		t.Errorf("not_found = %v", run.Result.NotFound)
	}
}
