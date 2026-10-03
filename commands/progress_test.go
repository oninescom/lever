package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressBarShowsActualCompletion(t *testing.T) {
	var output bytes.Buffer
	progress := &progressBar{label: "Copying", total: 100, output: &output}
	progress.Add(50)
	progress.Finish()
	if !strings.Contains(output.String(), " 50% 50B/100B") {
		t.Fatalf("partial progress missing: %q", output.String())
	}
	if strings.Contains(output.String(), "100%") {
		t.Fatalf("incomplete transfer reported complete: %q", output.String())
	}
	progress.Add(50)
	progress.Finish()
	if !strings.Contains(output.String(), "100% 100B/100B") {
		t.Fatalf("completion missing: %q", output.String())
	}
}

func TestProgressBarClearsShorterLine(t *testing.T) {
	var output bytes.Buffer
	progress := &progressBar{label: "Copying", total: 1 << 20, output: &output}
	progress.Add((1 << 20) - 1024)
	progress.Add(1024)
	progress.Finish()
	if !strings.Contains(output.String(), "100% 1.0M/1.0M ") {
		t.Fatalf("shorter line was not padded: %q", output.String())
	}
}
