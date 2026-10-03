package commands

import (
	"errors"
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
	"path/filepath"
	"syscall"
)

func NewMvCmd() *engine.Command {
	return &engine.Command{
		Use:   "mv [source...] [destination]",
		Short: "Linux 风格的 mv 移动/重命名工具",
		Long:  `移动或重命名文件/文件夹。支持多文件批量移动，跨盘符自动执行安全迁移降级。`,
		Args:  utils.MinimumNArgs(2),
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			sources := flags.Args()[:len(flags.Args())-1]
			dest := flags.Args()[len(flags.Args())-1]

			if len(sources) > 1 && !utils.ValidMultiSourceDestination(sources, dest) {
				return fmt.Errorf("mv 错误: 移动多个文件时，目标必须是一个已存在的目录")
			}

			var failures []error
			for _, src := range sources {
				finalDest := dest
				destStat, err := os.Stat(dest)
				if err == nil && destStat.IsDir() {
					finalDest = filepath.Join(dest, filepath.Base(src))
				}

				err = movePath(src, finalDest)
				if err != nil {
					failures = append(failures, fmt.Errorf("mv 错误: 移动 '%s' 失败: %w", src, err))
				}
			}
			return errors.Join(failures...)
		},
	}
}

func movePath(src, dest string) error {
	err := os.Rename(src, dest)
	if err == nil {
		return nil
	}
	// Windows 的 ERROR_NOT_SAME_DEVICE 表示跨卷移动；其他错误不能通过复制后删除绕过。
	if !errors.Is(err, syscall.Errno(17)) {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := checkCopyDestination(src, dest); err != nil {
			return err
		}
		err = copyDir(src, dest, true)
	} else {
		err = copyFile(src, dest, true)
	}
	if err != nil {
		return err
	}
	return os.RemoveAll(src)
}
