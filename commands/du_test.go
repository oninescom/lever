package commands

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestScanDiskUsageNestedDirectories(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	grandchild := filepath.Join(child, "grandchild")
	if err := os.MkdirAll(grandchild, 0755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, "root.txt"):       "a",
		filepath.Join(child, "child.txt"):     "bb",
		filepath.Join(grandchild, "deep.txt"): "ccc",
	} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := scanDiskUsage(root, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []diskUsage{{root, 6}, {child, 5}, {grandchild, 3}}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %#v", len(entries), len(want), entries)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Fatalf("entry %d = %#v, want %#v", i, entries[i], want[i])
		}
	}

	summary, err := scanDiskUsage(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 1 || summary[0] != want[0] {
		t.Fatalf("summary = %#v, want %#v", summary, want[0])
	}
}

func TestDuReportsMissingTarget(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if err := NewDuCmd().ExecuteArgs([]string{missing}); err == nil {
		t.Fatal("expected missing target to return an error")
	}
}

func TestDuSkipsProtectedDescendantsOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	child := filepath.Join(root, "protected")
	if err := duWalkError(root, child, fs.ErrPermission); err != nil {
		t.Fatalf("protected descendant should be skipped: %v", err)
	}
	if err := duWalkError(root, root, fs.ErrPermission); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("inaccessible target should return permission error: %v", err)
	}
	if err := duWalkError(root, child, fs.ErrNotExist); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("non-permission error should be returned: %v", err)
	}
}
