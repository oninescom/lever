package commands

import (
	"errors"
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
)

func NewMkdirCmd() *engine.Command {
	var parents bool
	var cmd = &engine.Command{
		Use:   "mkdir [directory...]",
		Short: "Linux 风格的 mkdir 工具",
		Long:  `创建目录。支持使用 -p 参数一键递归创建多级嵌套子目录。`,
		Args:  utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			var failures []error
			for _, dir := range flags.Args() {
				var err error
				if parents {
					err = os.MkdirAll(dir, 0755)
				} else {
					err = os.Mkdir(dir, 0755)
				}

				if err != nil {
					failures = append(failures, fmt.Errorf("mkdir 错误: 无法创建目录 '%s': %w", dir, err))
				}
			}
			return errors.Join(failures...)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&parents, "parents", "p", false, "递归创建多级目录，父目录不存在时自动创建")
	return cmd
}
