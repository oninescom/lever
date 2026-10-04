package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertProfileBlockReplacesOldVersion(t *testing.T) {
	existing := "before\n" + BlockStart + "\nold function\n" + BlockEnd + "\nafter\n"
	block := BlockStart + "\nnew function\n" + BlockEnd
	updated, err := upsertProfileBlock(existing, block)
	if err != nil {
		t.Fatal(err)
	}
	if updated != "before\n"+block+"\nafter\n" {
		t.Fatalf("unexpected profile content: %q", updated)
	}
	if strings.Count(updated, BlockStart) != 1 {
		t.Fatalf("duplicate Lever blocks: %q", updated)
	}
}

func TestUpsertProfileBlockRejectsIncompleteBlock(t *testing.T) {
	if _, err := upsertProfileBlock(BlockStart+"\nold function", BlockStart+"\nnew function\n"+BlockEnd); err == nil {
		t.Fatal("expected incomplete existing block to be rejected")
	}
}

func TestPathContainsExactDirectory(t *testing.T) {
	dir := `C:\Tools\Lever\bin`
	if pathContains(`C:\Tools\Lever\binary;C:\Other`, dir) {
		t.Fatal("similar directory must not match")
	}
	if !pathContains(`C:\Other;C:\Tools\Lever\bin\`, dir) {
		t.Fatal("same directory with trailing separator should match")
	}
}

func TestWriteShimPreservesForeignFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ls.cmd")
	if err := os.WriteFile(path, []byte("@echo off\r\necho mine\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeShim(path, `C:\Tools\lever.exe`, "ls"); err == nil {
		t.Fatal("expected existing foreign file to be preserved")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "@echo off\r\necho mine\r\n" {
		t.Fatalf("foreign file changed: %q", content)
	}
}

func TestRemoveLeverShimsRemovesEmptyDirectories(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Lever", "bin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeShim(filepath.Join(dir, "ls.cmd"), `C:\Tools\lever.exe`, "ls"); err != nil {
		t.Fatal(err)
	}
	if err := removeLeverShims(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "Lever")); !os.IsNotExist(err) {
		t.Fatalf("Lever directory still exists: %v", err)
	}
}

func TestRemoveLeverShimsPreservesForeignFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Lever", "bin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeShim(filepath.Join(dir, "ls.cmd"), `C:\Tools\lever.exe`, "ls"); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(foreign, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := removeLeverShims(dir); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(foreign); err != nil || string(data) != "keep" {
		t.Fatalf("foreign file changed: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ls.cmd")); !os.IsNotExist(err) {
		t.Fatalf("Lever shim still exists: %v", err)
	}
}

func TestShimForwardsArguments(t *testing.T) {
	got := shimContent(`C:\My Tools\lever.exe`, "awk")
	want := shimMarker + "\r\n@echo off\r\n@" + `"C:\My Tools\lever.exe" awk %*` + "\r\n"
	if got != want {
		t.Fatalf("unexpected shim: %q", got)
	}
}

func TestLegacyAutoRunDetection(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "cmd_init.bat")
	if err := os.WriteFile(legacy, []byte("@echo off\r\ndoskey ls=\"C:\\Old\\lever.exe\" ls $*\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !isLegacyAutoRun(legacy, `C:\New\lever.exe`) {
		t.Fatal("old Lever AutoRun should be recognized")
	}
	if err := os.WriteFile(legacy, []byte("@echo off\r\necho user script\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if isLegacyAutoRun(legacy, `C:\New\lever.exe`) {
		t.Fatal("unrelated AutoRun must be preserved")
	}
}

func TestCMDShimPipeline(t *testing.T) {
	dir := t.TempDir()
	exePath := filepath.Join(dir, "lever.exe")
	build := exec.Command("go", "build", "-o", exePath, "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build lever: %v\n%s", err, output)
	}
	shimPath := filepath.Join(dir, "awk.cmd")
	if err := writeShim(shimPath, exePath, "awk"); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("cmd.exe", "/d", "/c", "echo a b c|awk.cmd -1")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CMD pipeline: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "c" {
		t.Fatalf("unexpected pipeline output: %q", output)
	}
}

func TestCMDAutoRunPrependsShimDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "curl.cmd"), []byte("@echo off\r\necho lever-shim\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "check.cmd")
	if err := os.WriteFile(script, []byte("@echo off\r\n"+cmdAutoRunValue(dir, `C:\Tools\lever.exe`, []string{"curl"}, "")+"\r\ncurl\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("cmd.exe", "/d", "/c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CMD command lookup: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "lever-shim" {
		t.Fatalf("CMD did not prefer Lever shim: %q", output)
	}
	if got := cmdAutoRunValue(dir, `C:\Tools\lever.exe`, []string{"curl"}, "echo previous"); !strings.HasSuffix(got, " & echo previous") || !strings.Contains(got, `doskey curl="C:\Tools\lever.exe" curl $*`) {
		t.Fatalf("existing AutoRun was not preserved: %q", got)
	}
}
