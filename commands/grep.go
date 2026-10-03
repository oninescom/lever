package commands

import (
	"bufio"
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
		Short: "Linux 风格的 grep 文本搜索工具",
		Long:  `使用强大的正则表达式在本地文件或管道流中检索特定的文本行。支持高亮、计数、反选等主流功能。`,
		Args:  utils.MinimumNArgs(1),
		Run: func(c *engine.Command, args []string) {
			flags := c.Flags()
			patternStr := flags.Arg(0)
			files := flags.Args()[1:]

			if ignoreCase {
				patternStr = "(?i)" + patternStr
			}
			reg, err := regexp.Compile(patternStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "grep 错误: 无效的正则表达式 '%s': %v\n", patternStr, err)
				os.Exit(1)
			}
			if len(files) == 0 && utils.StdinIsPipe() {
				processGrepStream(os.Stdin, reg, "", invertMatch, lineNumber, countOnly, filesWithMatch, colorOpt)
				return
			}

			if len(files) == 0 {
				fmt.Println("grep 错误: 未指定输入文件或没有检测到管道输入")
				return
			}

			showFilename := len(files) > 1
			for _, filename := range files {
				file, err := os.Open(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "grep 错误: 无法打开文件 %s: %v\n", filename, err)
					continue
				}

				ctxName := ""
				if showFilename || filesWithMatch {
					ctxName = filename
				}

				processGrepStream(file, reg, ctxName, invertMatch, lineNumber, countOnly, filesWithMatch, colorOpt)
				file.Close()
			}
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&ignoreCase, "ignore-case", "i", false, "忽略大小写差异")
	flags.BoolVarP(&invertMatch, "invert-match", "v", false, "反向选择，只显示不匹配的行")
	flags.BoolVarP(&lineNumber, "line-number", "n", false, "在输出前面加上行号")
	flags.BoolVarP(&countOnly, "count", "c", false, "只计算并打印匹配成功的行数")
	flags.BoolVarP(&filesWithMatch, "files-with-matches", "l", false, "只列出匹配成功的文件名，而不显示具体内容")
	flags.StringVar(&colorOpt, "color", "auto", "高亮显示匹配的关键词 (always, never, auto)")
	return cmd

}

func processGrepStream(reader io.Reader, reg *regexp.Regexp, filename string, invert, showLine, countMode, listFiles bool, color string) {
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
				return
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
					// 优雅跨包调用全局唯一的 utils 红色与重置常量
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
}
