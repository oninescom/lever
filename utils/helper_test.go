package utils

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPowerShellProfileWritePreservesBytes(t *testing.T) {
	if _, err := exec.LookPath("pwsh"); err != nil {
		t.Skip("pwsh is not available")
	}
	path := filepath.Join(t.TempDir(), "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	content := []byte{0xff, 0xfe, 'a', 0, '\n', 0}
	if err := writeProfileWithPowerShell(path, content); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, content) {
		t.Fatalf("profile bytes changed: %v", actual)
	}
}
