package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
	"path/filepath"
)

func NewMvCmd() *engine.Command {
	return &engine.Command{
		Use:   "mv [source...] [destination]",
		Short: "Linux 风格的 mv 移动/重命名工具",
		Long:  `移动或重命名文件/文件夹。支持多文件批量移动，跨盘符自动执行安全迁移降级。`,
		Args:  utils.MinimumNArgs(2),
		Run: func(c *engine.Command, args []string) {
			flags := c.Flags()
			sources := flags.Args()[:len(flags.Args())-1]
			dest := flags.Args()[len(flags.Args())-1]

			if len(sources) > 1 && !utils.ValidMultiSourceDestination(sources, dest) {
				fmt.Fprintln(os.Stderr, "mv 错误: 移动多个文件时，目标必须是一个已存在的目录")
				os.Exit(1)
			}

			for _, src := range sources {
				finalDest := dest
				destStat, err := os.Stat(dest)
				if err == nil && destStat.IsDir() {
					finalDest = filepath.Join(dest, filepath.Base(src))
				}

				// 1. 优先尝试原子级系统重命名（同盘符下 0 毫秒级秒切）
				err = os.Rename(src, finalDest)
				if err != nil {
					// 2. 如果遇到跨盘符报错，自动安全降级处理：深度复制 + 强力擦除
					err = copyPath(src, finalDest, true, true)
					if err == nil {
						err = os.RemoveAll(src)
					}
				}

				if err != nil {
					fmt.Fprintf(os.Stderr, "mv 错误: 移动 '%s' 失败: %v\n", src, err)
				}
			}
		},
	}
}
