package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestAwkNegativeIndices(t *testing.T) {
	tests := []struct {
		args  []string
		input string
		want  string
	}{
		{[]string{"-1"}, "a b c\n", "c\n"},
		{[]string{"1", "-1"}, "a b c\n", "a c\n"},
		{[]string{"-F", ",", "-1"}, "a,b,c\n", "c\n"},
		{[]string{"1", "-F", ",", "-1"}, "a,b,c\n", "a c\n"},
		{[]string{"-F", "-1", "1"}, "a-1b\n", "a\n"},
		{[]string{"--", "-1"}, "a b c\n", "c\n"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, "_"), func(t *testing.T) {
			stdin, input, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout, output, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer stdin.Close()
			defer stdout.Close()
			if _, err := io.WriteString(input, tt.input); err != nil {
				t.Fatal(err)
			}
			input.Close()

			oldStdin, oldStdout := os.Stdin, os.Stdout
			os.Stdin, os.Stdout = stdin, output
			err = NewAwkCmd().ExecuteArgs(tt.args)
			os.Stdin, os.Stdout = oldStdin, oldStdout
			output.Close()
			if err != nil {
				t.Fatal(err)
			}
			actual, err := io.ReadAll(stdout)
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != tt.want {
				t.Fatalf("got %q, want %q", actual, tt.want)
			}
		})
	}
}

func TestAwkLongLineWithoutTrailingNewline(t *testing.T) {
	var output bytes.Buffer
	input := strings.Repeat("x", 70_000) + " last"
	if err := processAwkStream(strings.NewReader(input), &output, "", []int{-1}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "last\n" {
		t.Fatalf("got %q, want %q", got, "last\n")
	}
}
