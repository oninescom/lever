package commands

import (
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
	"path/filepath"
)

func NewCpCmd() *engine.Command {
	var recursive bool
	var force bool

	cmd := &engine.Command{
		Use:   "cp [source...] [destination]",
		Short: "Linux 风格的 cp 复制工具",
		Long:  `复制文件或文件夹。支持使用 -r 递归复制整个目录树，以及使用 -f 强制覆盖。`,
		Args:  utils.MinimumNArgs(2),
		Run: func(c *engine.Command, args []string) {
			flags := c.Flags()
			sources := flags.Args()[:len(flags.Args())-1]
			dest := flags.Args()[len(flags.Args())-1]

			// 多文件批量复制时，利用你上传的 ValidMultiSourceDestination 进行目标目录校验
			if len(sources) > 1 && !utils.ValidMultiSourceDestination(sources, dest) {
				fmt.Fprintln(os.Stderr, "cp 错误: 复制多个文件或目录时，目标必须是一个已存在的目录")
				os.Exit(1)
			}

			for _, src := range sources {
				err := copyPath(src, dest, recursive, force)
				if err != nil {
					fmt.Fprintf(os.Stderr, "cp 错误: 复制 '%s' 失败: %v\n", src, err)
				}
			}
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&recursive, "recursive", "r", false, "递归复制目录下的所有内容")
	flags.BoolVarP(&force, "force", "f", false, "强制覆盖已存在的目标文件")

	return cmd
}

func copyPath(src, dest string, recursive, force bool) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	// 如果目标本身就是已存在目录，则在内部新建同名文件
	destStat, err := os.Stat(dest)
	if err == nil && destStat.IsDir() {
		dest = filepath.Join(dest, filepath.Base(src))
	}

	if srcStat.IsDir() {
		if !recursive {
			return fmt.Errorf("'%s' 是一个目录 (未指定 -r 参数)", src)
		}
		return copyDir(src, dest, force)
	}
	return copyFile(src, dest, force)
}

func copyFile(src, dest string, force bool) error {
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("目标文件 '%s' 已存在 (可使用 -f 强制覆盖)", dest)
		}
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(srcDir, destDir string, force bool) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		var err error
		if entry.IsDir() {
			err = copyDir(srcPath, destPath, force)
		} else {
			err = copyFile(srcPath, destPath, force)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
