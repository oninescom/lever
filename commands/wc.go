package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
	"strings"
)

func NewWcCmd() *engine.Command {
	var countLines, countWords, countBytes bool

	cmd := &engine.Command{
		Use:   "wc [file...]",
		Short: "Linux-style wc word, line, and byte count utility",
		Long:  `Print newline, word, and byte counts for each FILE, and a total line if more than one FILE is specified. Supports pipeline stream processing perfectly.`,
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			files := flags.Args()

			// If no files specified and stdin is a pipe, stream process stdin
			if len(files) == 0 && utils.StdinIsPipe() {
				lines, words, bytes, err := processWcStream(os.Stdin)
				if err != nil {
					return err
				}
				printWcResult(lines, words, bytes, "", countLines, countWords, countBytes)
				return nil
			}

			if len(files) == 0 {
				return fmt.Errorf("wc error: missing file operand and no incoming pipeline stream detected")
			}

			var totalLines, totalWords, totalBytes int64
			var failures []error
			for _, filename := range files {
				file, err := os.Open(filename)
				if err != nil {
					failures = append(failures, fmt.Errorf("wc error: cannot open file %s: %w", filename, err))
					continue
				}
				l, w, b, err := processWcStream(file)
				closeErr := file.Close()
				if err != nil {
					failures = append(failures, fmt.Errorf("wc error: cannot read file %s: %w", filename, err))
					continue
				}
				if closeErr != nil {
					failures = append(failures, fmt.Errorf("wc error: cannot close file %s: %w", filename, closeErr))
					continue
				}
				printWcResult(l, w, b, filename, countLines, countWords, countBytes)
				totalLines += l
				totalWords += w
				totalBytes += b
			}

			if len(files) > 1 {
				printWcResult(totalLines, totalWords, totalBytes, "total", countLines, countWords, countBytes)
			}
			return errors.Join(failures...)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&countLines, "lines", "l", false, "print the newline counts")
	flags.BoolVarP(&countWords, "words", "w", false, "print the word counts")
	flags.BoolVarP(&countBytes, "bytes", "c", false, "print the byte counts")
	return cmd
}

func processWcStream(r io.Reader) (lines, words, bytes int64, err error) {
	bufReader := bufio.NewReader(r)
	inWord := false

	for {
		line, readErr := bufReader.ReadString('\n')
		lineLen := int64(len(line))
		bytes += lineLen

		if lineLen > 0 {
			if strings.HasSuffix(line, "\n") {
				lines++
			}
			// Word state boundaries validation machine
			for _, char := range line {
				if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
					inWord = false
				} else if !inWord {
					inWord = true
					words++
				}
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return 0, 0, 0, readErr
		}
	}
	return lines, words, bytes, nil
}

func printWcResult(l, w, b int64, name string, cl, cw, cb bool) {
	if !cl && !cw && !cb {
		cl, cw, cb = true, true, true
	}

	var out []string
	if cl {
		out = append(out, fmt.Sprintf("%7d", l))
	}
	if cw {
		out = append(out, fmt.Sprintf("%7d", w))
	}
	if cb {
		out = append(out, fmt.Sprintf("%7d", b))
	}
	if name != "" {
		out = append(out, name)
	}

	fmt.Println(strings.Join(out, " "))
}
