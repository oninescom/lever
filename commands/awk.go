package commands

import (
	"bufio"
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
	"strconv"
	"strings"
)

func NewAwkCmd() *engine.Command {
	var fieldDelimiter string

	cmd := &engine.Command{
		Use:         "awk [column_indices...]",
		Short:       `Select columns from text streams`,
		Long:        `Read piped input and select columns from each line. Column indices start at 1; -1 selects the last column.`,
		PrepareArgs: awkFlagArgs,
		// Require at least one column index.
		Args: utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {

			// Parse requested column indices.
			var targetCols []int
			for _, str := range args {
				idx, err := strconv.Atoi(str)
				if err != nil {
					return fmt.Errorf("awk error: invalid column index '%s'; expected an integer", str)
				}
				targetCols = append(targetCols, idx)
			}

			// Read from piped standard input.
			if utils.StdinIsPipe() {
				return processAwkStream(os.Stdin, os.Stdout, fieldDelimiter, targetCols)
			} else {
				return fmt.Errorf("awk error: no piped input detected. Example: ls | awk 1 5")
			}
		},
	}

	flags := cmd.Flags()
	// -F selects a custom field separator.
	flags.StringVarP(&fieldDelimiter, "field-separator", "F", "", "Field separator (default: consecutive whitespace)")

	return cmd
}

func awkFlagArgs(args []string) []string {
	var flags, indices []string
	positionalOnly := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if positionalOnly {
			indices = append(indices, arg)
			continue
		}
		switch {
		case arg == "--":
			positionalOnly = true
		case arg == "-F" || arg == "--field-separator":
			flags = append(flags, arg)
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		case arg == "-h" || arg == "--help" || strings.HasPrefix(arg, "-F") || strings.HasPrefix(arg, "--field-separator="):
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "-") && !isNegativeIndex(arg):
			flags = append(flags, arg)
		default:
			indices = append(indices, arg)
		}
	}
	if len(indices) == 0 {
		return flags
	}
	return append(flags, append([]string{"--"}, indices...)...)
}

func isNegativeIndex(value string) bool {
	return len(value) > 1 && value[0] == '-' && strings.Trim(value[1:], "0123456789") == ""
}

// processAwkStream scans each input line and selects the requested fields.
func processAwkStream(reader io.Reader, writer io.Writer, delimiter string, targetCols []int) error {
	input := bufio.NewReader(reader)
	for {
		line, readErr := input.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		if readErr == io.EOF && line == "" {
			return nil
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		var columns []string

		// Whitespace mode treats consecutive whitespace as one separator.
		if delimiter == "" {
			columns = strings.Fields(line)
		} else {
			// A custom separator splits at each occurrence.
			columns = strings.Split(line, delimiter)
		}

		// Skip lines with no fields.
		if len(columns) == 0 {
			if readErr == io.EOF {
				return nil
			}
			continue
		}

		// Select fields using one-based or negative indices.
		var result []string
		for _, colIdx := range targetCols {
			actualIdx := 0

			if colIdx > 0 {
				// 1 selects the first field.
				actualIdx = colIdx - 1
			} else if colIdx < 0 {
				// -1 selects the last field.
				actualIdx = len(columns) + colIdx
			} else {
				// 0 selects the entire line.
				result = append(result, line)
				continue
			}

			// Out-of-range fields contribute an empty string.
			if actualIdx >= 0 && actualIdx < len(columns) {
				result = append(result, columns[actualIdx])
			} else {
				result = append(result, "")
			}
		}

		if _, err := fmt.Fprintln(writer, strings.Join(result, " ")); err != nil {
			return err
		}
		if readErr == io.EOF {
			return nil
		}
	}
}
