package commands

import (
	"bufio"
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
	"regexp"
	"strings"
)

func NewSedCmd() *engine.Command {
	cmd := &engine.Command{
		Use:   "sed [s/pattern/replacement/g]",
		Short: "Stream editor for non-interactive text pipeline regex replacement",
		Long:  `Intercepts incoming pipelined data and maps global substitutions. Specification: s/match_pattern/replace_token/g`,
		Args:  utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {
			expr := args[0]
			patternStr, replaceStr, globalMode, err := parseSedExpression(expr)
			if err != nil {
				return err
			}

			reg, err := regexp.Compile(patternStr)
			if err != nil {
				return fmt.Errorf("sed error: failed to compile target regex pattern: %v", err)
			}

			// Check and capture continuous pipeline data streams safely
			if utils.StdinIsPipe() {
				bufReader := bufio.NewReader(os.Stdin)
				for {
					line, readErr := bufReader.ReadString('\n')
					if line != "" {
						cleanLine := strings.TrimRight(line, "\r\n")

						var outLine string
						if globalMode {
							// Global replacement across the entire string block
							outLine = reg.ReplaceAllString(cleanLine, replaceStr)
						} else {
							// Match-first replacement mode (substitutes only the leading keyword match)
							loc := reg.FindStringIndex(cleanLine)
							if loc != nil {
								outLine = cleanLine[:loc[0]] + replaceStr + cleanLine[loc[1]:]
							} else {
								outLine = cleanLine
							}
						}
						fmt.Println(outLine)
					}

					if readErr != nil {
						if readErr == io.EOF {
							break // Pipeline stream completely drained
						}
						return readErr
					}
				}
			} else {
				return fmt.Errorf("sed error: no incoming pipeline data flow detected. Example: cat file | sed s/old/new/g")
			}
			return nil
		},
	}
	return cmd
}

func parseSedExpression(expr string) (pattern, replacement string, global bool, err error) {
	if !strings.HasPrefix(expr, "s/") {
		return "", "", false, fmt.Errorf("sed error: expression must start with 's/'")
	}
	parts := make([]string, 0, 2)
	var current strings.Builder
	for i := 2; i < len(expr); i++ {
		switch {
		case expr[i] == '\\' && i+1 < len(expr) && expr[i+1] == '/':
			current.WriteByte('/')
			i++
		case expr[i] == '\\' && i+1 < len(expr) && expr[i+1] == '\\':
			current.WriteString(`\\`)
			i++
		case expr[i] == '/':
			parts = append(parts, current.String())
			current.Reset()
			if len(parts) == 2 {
				suffix := expr[i+1:]
				if suffix != "" && suffix != "g" {
					return "", "", false, fmt.Errorf("sed error: unsupported expression suffix %q", suffix)
				}
				return parts[0], parts[1], suffix == "g", nil
			}
		default:
			current.WriteByte(expr[i])
		}
	}
	return "", "", false, fmt.Errorf("sed error: incomplete expression; expected s/pattern/replacement/[g]")
}
