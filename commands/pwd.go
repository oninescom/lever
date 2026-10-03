package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
)

func NewPwdCmd() *engine.Command {
	return &engine.Command{
		Use:   "pwd",
		Short: "Linux 风格的 pwd 路径工具",
		Long:  `打印当前工作目录的完整绝对物理路径。`,
		Args:  utils.NoArgs,
		Run: func(c *engine.Command, args []string) {
			dir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "pwd 错误: 无法获取当前工作路径: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(dir)
		},
	}
}
