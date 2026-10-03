package commands

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

func TestTailCLI(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "lever.exe")
	build := exec.Command("go", "build", "-o", exe, "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build lever: %v\n%s", err, output)
	}
	for _, tc := range []struct {
		name, input, want string
		args              []string
	}{
		{"trailing newline", "a\nb\nc\n", "b\nc\n", []string{"-2"}},
		{"no trailing newline", "a\nb\nc", "b\nc", []string{"-n", "2"}},
		{"zero lines", "a\nb\n", "", []string{"-n", "0"}},
		{"larger than block", strings.Repeat("x\n", 3000), "x\nx\n", []string{"-2"}},
		{"across blocks", strings.Repeat("x\n", 3000), strings.Repeat("x\n", 2500), []string{"-2500"}},
		{"long line", "first\n" + strings.Repeat("x", 10000), strings.Repeat("x", 10000), []string{"-1"}},
		{"empty file", "", "", []string{"-2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(dir, tc.name+".txt")
			if err := os.WriteFile(file, []byte(tc.input), 0644); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"tail"}, tc.args...)
			cmd := exec.Command(exe, append(args, file)...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("tail failed: %v\n%s", err, output)
			}
			if string(output) != tc.want {
				t.Fatalf("got %q, want %q", output, tc.want)
			}
		})
	}
	t.Run("follow append and truncate", func(t *testing.T) {
		file := filepath.Join(dir, "follow.txt")
		if err := os.WriteFile(file, []byte("seed\n"), 0644); err != nil {
			t.Fatal(err)
		}
		outputPath := filepath.Join(dir, "follow-output.txt")
		outputFile, err := os.Create(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		defer outputFile.Close()
		cmd := exec.Command(exe, "tail", "-f", "-n", "0", file)
		cmd.Stdout = outputFile
		cmd.Stderr = outputFile
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}()
		time.Sleep(200 * time.Millisecond)
		appendFile, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := appendFile.WriteString("append\n"); err != nil {
			t.Fatal(err)
		}
		appendFile.Close()
		waitTailOutput(t, outputPath, "append\n")
		if err := os.WriteFile(file, []byte("new\n"), 0644); err != nil {
			t.Fatal(err)
		}
		waitTailOutput(t, outputPath, "append\nnew\n")
	})
	for _, tc := range []struct {
		name, input, want string
		order             binary.ByteOrder
		bom               []byte
	}{
		{"UTF-16 LE long line", strings.Repeat("A", 100000) + " -> LINE_END\r\n", strings.Repeat("A", 100000) + " -> LINE_END\r\n", binary.LittleEndian, []byte{0xff, 0xfe}},
		{"UTF-16 BE last line", "first\nsecond 😀\n", "second 😀\n", binary.BigEndian, []byte{0xfe, 0xff}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(dir, tc.name+".txt")
			var raw bytes.Buffer
			raw.Write(tc.bom)
			for _, unit := range utf16.Encode([]rune(tc.input)) {
				if err := binary.Write(&raw, tc.order, unit); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(file, raw.Bytes(), 0644); err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command(exe, "tail", "-1", file).CombinedOutput()
			if err != nil {
				t.Fatalf("tail failed: %v\n%s", err, output)
			}
			if string(output) != tc.want {
				t.Fatalf("got %d bytes, want %d bytes; suffix %q", len(output), len(tc.want), string(output[max(0, len(output)-32):]))
			}
		})
	}
}

func waitTailOutput(t *testing.T, path, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(content), want) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	content, _ := os.ReadFile(path)
	t.Fatalf("tail -f output %q does not contain %q", content, want)
}
