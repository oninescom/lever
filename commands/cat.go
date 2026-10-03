package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
)

func NewCatCmd() *engine.Command {
	var numberLines bool

	var cmd = &engine.Command{
		Use:   "cat [file...]",
		Short: "Print file contents",
		Long:  `Print one or more files to standard output. Use -n to number lines; piped input is also supported.`,
		RunE: func(cmd *engine.Command, args []string) error {
			flags := cmd.Flags()
			if flags.NArg() == 0 && utils.StdinIsPipe() {
				return readSource(os.Stdin, numberLines)
			}
			if flags.NArg() == 0 {
				return cmd.Help()
			}

			var failures []error
			for _, filename := range args {
				file, err := os.Open(filename)
				if err != nil {
					failures = append(failures, fmt.Errorf("cat error: cannot open %s: %w", filename, err))
					continue
				}
				if err := readSource(file, numberLines); err != nil {
					failures = append(failures, fmt.Errorf("cat error: cannot read %s: %w", filename, err))
				}
				if err := file.Close(); err != nil {
					failures = append(failures, fmt.Errorf("cat error: cannot close %s: %w", filename, err))
				}
			}
			return errors.Join(failures...)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&numberLines, "number", "n", false, "Number all output lines")
	return cmd
}

func readSource(reader io.Reader, numberLines bool) error {
	if !numberLines {
		_, err := io.Copy(os.Stdout, reader)
		return err
	} else {
		scanner := bufio.NewScanner(reader)
		lineNumber := 1
		for scanner.Scan() {
			fmt.Printf("%6d  %s\n", lineNumber, scanner.Text())
			lineNumber++
		}
		return scanner.Err()
	}
}
