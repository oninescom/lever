package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
	"regexp"
)

func NewGrepCmd() *engine.Command {
	var ignoreCase, invertMatch, lineNumber, countOnly, filesWithMatch bool
	var colorOpt string

	cmd := &engine.Command{
		Use:   "grep [pattern] [file...]",
		Short: "Search text with regular expressions",
		Long:  `Search files or piped input with regular expressions. Supports highlighting, counting, and inverted matches.`,
		Args:  utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			patternStr := flags.Arg(0)
			files := flags.Args()[1:]

			if ignoreCase {
				patternStr = "(?i)" + patternStr
			}
			reg, err := regexp.Compile(patternStr)
			if err != nil {
				return fmt.Errorf("grep error: invalid regular expression '%s': %w", patternStr, err)
			}
			if len(files) == 0 && utils.StdinIsPipe() {
				return processGrepStream(os.Stdin, reg, "", invertMatch, lineNumber, countOnly, filesWithMatch, colorOpt)
			}

			if len(files) == 0 {
				return fmt.Errorf("grep error: no input file or piped input provided")
			}

			showFilename := len(files) > 1
			var failures []error
			for _, filename := range files {
				file, err := os.Open(filename)
				if err != nil {
					failures = append(failures, fmt.Errorf("grep error: cannot open %s: %w", filename, err))
					continue
				}

				ctxName := ""
				if showFilename || filesWithMatch {
					ctxName = filename
				}

				if err := processGrepStream(file, reg, ctxName, invertMatch, lineNumber, countOnly, filesWithMatch, colorOpt); err != nil {
					failures = append(failures, fmt.Errorf("grep error: cannot read %s: %w", filename, err))
				}
				if err := file.Close(); err != nil {
					failures = append(failures, fmt.Errorf("grep error: cannot close %s: %w", filename, err))
				}
			}
			return errors.Join(failures...)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&ignoreCase, "ignore-case", "i", false, "Ignore case")
	flags.BoolVarP(&invertMatch, "invert-match", "v", false, "Show only non-matching lines")
	flags.BoolVarP(&lineNumber, "line-number", "n", false, "Prefix output with line numbers")
	flags.BoolVarP(&countOnly, "count", "c", false, "Print only the number of matching lines")
	flags.BoolVarP(&filesWithMatch, "files-with-matches", "l", false, "Print only names of files with matches")
	flags.StringVar(&colorOpt, "color", "auto", "Highlight matches (always, never, auto)")
	return cmd

}

func processGrepStream(reader io.Reader, reg *regexp.Regexp, filename string, invert, showLine, countMode, listFiles bool, color string) error {
	scanner := bufio.NewScanner(reader)
	currentLine := 0
	matchCount := 0

	for scanner.Scan() {
		currentLine++
		text := scanner.Text()
		hasMatch := reg.MatchString(text)

		if (hasMatch && !invert) || (!hasMatch && invert) {
			matchCount++

			if listFiles {
				if filename != "" {
					fmt.Println(filename)
				}
				return nil
			}

			if !countMode {
				prefix := ""
				if filename != "" && !listFiles {
					prefix += filename + ":"
				}
				if showLine {
					prefix += fmt.Sprintf("%d:", currentLine)
				}

				outText := text
				if !invert && (color == "always" || color == "auto") {
					outText = reg.ReplaceAllStringFunc(text, func(m string) string {
						return utils.ColorBoldRed + m + utils.ColorReset
					})
				}
				fmt.Printf("%s%s\n", prefix, outText)
			}
		}
	}

	if countMode {
		if filename != "" {
			fmt.Printf("%s:%d\n", filename, matchCount)
		} else {
			fmt.Println(matchCount)
		}
	}
	return scanner.Err()
}
