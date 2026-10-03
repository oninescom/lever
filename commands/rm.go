package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
)

func NewRmCmd() *engine.Command {
	var recursive bool
	var force bool

	cmd := &engine.Command{
		Use:   "rm [file...]",
		Short: "Linux 风格的 rm 删除工具",
		Long:  `删除指定的文件或目录。支持使用 -r 递归抹除目录，以及使用 -f 强行忽略不存在的文件。`,
		Args:  utils.MinimumNArgs(1),
		Run: func(c *engine.Command, args []string) {
			flags := c.Flags()

			for _, target := range flags.Args() {
				stat, err := os.Stat(target)
				if os.IsNotExist(err) {
					if force {
						continue // -f 强力模式下，目标不存在直接闭嘴跳过
					}
					fmt.Fprintf(os.Stderr, "rm 错误: 无法删除 '%s': 文件或目录不存在\n", target)
					continue
				}

				if stat.IsDir() && !recursive {
					fmt.Fprintf(os.Stderr, "rm 错误: 无法删除 '%s': 是一个目录 (未指定 -r 参数)\n", target)
					continue
				}

				// 使用 os.RemoveAll 原生打穿递归和文件锁定限制
				err = os.RemoveAll(target)
				if err != nil {
					fmt.Fprintf(os.Stderr, "rm 错误: 抹除 %s 失败: %v\n", target, err)
				}
			}
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&recursive, "recursive", "r", false, "递归删除目录及其下所有内容")
	flags.BoolVarP(&force, "force", "f", false, "强制删除，忽略不存在的文件且不报警告")

	return cmd
}
