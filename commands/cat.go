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
		Short: "Linux 风格的 cat 工具",
		Long:  `读取一个或多个文件的内容并打印到标准输出。支持使用 -n 进行行号排版，并完美支持管道符 (|) 输入。`,
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
					failures = append(failures, fmt.Errorf("cat 错误: 无法打开文件 %s: %w", filename, err))
					continue
				}
				if err := readSource(file, numberLines); err != nil {
					failures = append(failures, fmt.Errorf("cat 错误: 无法读取文件 %s: %w", filename, err))
				}
				if err := file.Close(); err != nil {
					failures = append(failures, fmt.Errorf("cat 错误: 无法关闭文件 %s: %w", filename, err))
				}
			}
			return errors.Join(failures...)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&numberLines, "number", "n", false, "对所有输出行进行编号")
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
