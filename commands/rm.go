package commands

import (
	"errors"
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
	"path/filepath"
)

func NewRmCmd() *engine.Command {
	var recursive bool
	var force bool

	cmd := &engine.Command{
		Use:   "rm [file...]",
		Short: "Linux 风格的 rm 删除工具",
		Long:  `删除指定的文件或目录。支持使用 -r 递归抹除目录，以及使用 -f 强行忽略不存在的文件。`,
		Args:  utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			var failures []error

			for _, target := range flags.Args() {
				stat, err := os.Stat(target)
				if os.IsNotExist(err) {
					if force {
						continue
					}
					failures = append(failures, fmt.Errorf("rm 错误: 无法删除 '%s': 文件或目录不存在", target))
					continue
				}
				if err != nil {
					failures = append(failures, fmt.Errorf("rm 错误: 无法检查 '%s': %w", target, err))
					continue
				}

				if stat.IsDir() && !recursive {
					failures = append(failures, fmt.Errorf("rm 错误: 无法删除 '%s': 是一个目录 (未指定 -r 参数)", target))
					continue
				}

				if force {
					_ = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
						if err == nil {
							_ = os.Chmod(path, 0666)
						}
						return nil
					})
				}

				err = os.RemoveAll(target)
				if err != nil {
					failures = append(failures, fmt.Errorf("rm 错误: 抹除 %s 失败: %w", target, err))
				}
			}
			return errors.Join(failures...)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&recursive, "recursive", "r", false, "递归删除目录及其下所有内容")
	flags.BoolVarP(&force, "force", "f", false, "强制删除，忽略不存在的文件且不报警告")

	return cmd
}
