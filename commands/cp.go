package commands

import (
	"errors"
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
	"path/filepath"
	"strings"
)

func NewCpCmd() *engine.Command {
	var recursive bool
	var force bool

	cmd := &engine.Command{
		Use:   "cp [source...] [destination]",
		Short: "Linux 风格的 cp 复制工具",
		Long:  `复制文件或文件夹。支持使用 -r 递归复制整个目录树，以及使用 -f 强制覆盖。`,
		Args:  utils.MinimumNArgs(2),
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			sources := flags.Args()[:len(flags.Args())-1]
			dest := flags.Args()[len(flags.Args())-1]

			// 多文件批量复制时，利用你上传的 ValidMultiSourceDestination 进行目标目录校验
			if len(sources) > 1 && !utils.ValidMultiSourceDestination(sources, dest) {
				return fmt.Errorf("cp 错误: 复制多个文件或目录时，目标必须是一个已存在的目录")
			}

			var failures []error
			for _, src := range sources {
				err := copyPath(src, dest, recursive, force)
				if err != nil {
					failures = append(failures, fmt.Errorf("cp 错误: 复制 '%s' 失败: %w", src, err))
				}
			}
			return errors.Join(failures...)
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
		if err := checkCopyDestination(src, dest); err != nil {
			return err
		}
		return copyDir(src, dest, force)
	}
	if err == nil && os.SameFile(srcStat, destStat) {
		return fmt.Errorf("源文件和目标文件相同")
	}
	return copyFile(src, dest, force)
}

func checkCopyDestination(src, dest string) error {
	source, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return err
	}

	target, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	var suffix []string
	for {
		resolved, err := filepath.EvalSymlinks(target)
		if err == nil {
			target = filepath.Join(append([]string{resolved}, suffix...)...)
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(target)
		if parent == target {
			return err
		}
		suffix = append([]string{filepath.Base(target)}, suffix...)
		target = parent
	}

	rel, err := filepath.Rel(source, target)
	if err != nil {
		return err
	}
	if rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("目标目录位于源目录内部: %s", dest)
	}
	return nil
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
