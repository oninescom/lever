package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
)

func NewClearCmd() *engine.Command {
	return &engine.Command{
		Use:   "clear",
		Short: "Linux 风格的 clear 清屏工具",
		Long:  `清空当前终端屏幕上的所有内容，并将光标重置到左上角。`,
		Args:  utils.NoArgs,
		Run: func(c *engine.Command, args []string) {
			fmt.Print("\033[H\033[2J")
		},
	}
}
