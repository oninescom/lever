package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyRejectsDestinationInsideSource(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	if err := os.Mkdir(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "data.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, dest := range []string{src, filepath.Join(src, "nested", "copy")} {
		if err := NewCpCmd().ExecuteArgs([]string{"-r", src, dest}); err == nil {
			t.Fatalf("expected error when copying a directory into %s", dest)
		}
	}
	if _, err := os.Stat(filepath.Join(src, "nested")); !os.IsNotExist(err) {
		t.Fatalf("copy created a destination inside source: %v", err)
	}
}

func TestCopyRejectsSameFileWithForce(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(file, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := NewCpCmd().ExecuteArgs([]string{"-f", file, file}); err == nil {
		t.Fatal("expected error when copying a file onto itself")
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "keep" {
		t.Fatalf("source content changed: %q, %v", data, err)
	}
}

func TestMoveRenameFailureKeepsSource(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source")
	dest := filepath.Join(root, "occupied")
	for _, dir := range []string{src, dest} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(src, "source.txt"), []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "existing.txt"), []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := movePath(src, dest); err == nil {
		t.Fatal("expected rename into occupied directory to fail")
	}
	if _, err := os.Stat(filepath.Join(src, "source.txt")); err != nil {
		t.Fatalf("source was removed after rename failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "source.txt")); !os.IsNotExist(err) {
		t.Fatalf("source was copied after rename failure: %v", err)
	}
}

func TestCopyContinuesAndReportsPartialFailure(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "good.txt")
	missing := filepath.Join(root, "missing.txt")
	dest := filepath.Join(root, "out")
	if err := os.WriteFile(good, []byte("good"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatal(err)
	}
	if err := NewCpCmd().ExecuteArgs([]string{missing, good, dest}); err == nil {
		t.Fatal("expected partial failure to be returned")
	}
	if _, err := os.Stat(filepath.Join(dest, "good.txt")); err != nil {
		t.Fatalf("valid source was not copied: %v", err)
	}
}

func TestRemoveContinuesAndReportsPartialFailure(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "good.txt")
	missing := filepath.Join(root, "missing.txt")
	if err := os.WriteFile(good, []byte("good"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := NewRmCmd().ExecuteArgs([]string{missing, good}); err == nil {
		t.Fatal("expected partial failure to be returned")
	}
	if _, err := os.Stat(good); !os.IsNotExist(err) {
		t.Fatalf("valid target was not removed: %v", err)
	}
}
