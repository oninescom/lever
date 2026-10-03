package commands

import (
	"bufio"
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
		Short: "Linux 风格的 cat 工具",
		Long:  `读取一个或多个文件的内容并打印到标准输出。支持使用 -n 进行行号排版，并完美支持管道符 (|) 输入。`,
		Run: func(cmd *engine.Command, args []string) {
			flags := cmd.Flags()
			if flags.NArg() == 0 && utils.StdinIsPipe() {
				readSource(os.Stdin, numberLines)
				return
			}
			if flags.NArg() == 0 {
				_ = cmd.Help()
				return
			}

			for _, filename := range args {
				file, err := os.Open(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "cat 错误: 无法打开文件 %s: %v\n", filename, err)
					continue
				}
				readSource(file, numberLines)
				file.Close()
			}
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&numberLines, "number", "n", false, "对所有输出行进行编号")
	return cmd
}

func readSource(reader io.Reader, numberLines bool) {
	if !numberLines {
		_, _ = io.Copy(os.Stdout, reader)
	} else {
		scanner := bufio.NewScanner(reader)
		lineNumber := 1
		for scanner.Scan() {
			fmt.Printf("%6d  %s\n", lineNumber, scanner.Text())
			lineNumber++
		}
	}
}
