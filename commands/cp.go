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
		Short: "Copy files and directories",
		Long:  `Copy files or directories. Use -r to copy directories recursively and -f to overwrite existing files.`,
		Args:  utils.MinimumNArgs(2),
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			sources := flags.Args()[:len(flags.Args())-1]
			dest := flags.Args()[len(flags.Args())-1]

			if len(sources) > 1 && !utils.ValidMultiSourceDestination(sources, dest) {
				return fmt.Errorf("cp error: destination must be an existing directory when copying multiple sources")
			}

			var failures []error
			for _, src := range sources {
				err := copyPath(src, dest, recursive, force)
				if err != nil {
					failures = append(failures, fmt.Errorf("cp error: cannot copy '%s': %w", src, err))
				}
			}
			return errors.Join(failures...)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&recursive, "recursive", "r", false, "Copy directories and their contents recursively")
	flags.BoolVarP(&force, "force", "f", false, "Overwrite existing destination files")

	return cmd
}

func copyPath(src, dest string, recursive, force bool) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	destStat, err := os.Stat(dest)
	if err == nil && destStat.IsDir() {
		dest = filepath.Join(dest, filepath.Base(src))
	}

	if srcStat.IsDir() {
		if !recursive {
			return fmt.Errorf("'%s' is a directory (use -r)", src)
		}
		if err := checkCopyDestination(src, dest); err != nil {
			return err
		}
		return copyDir(src, dest, force)
	}
	if err == nil && os.SameFile(srcStat, destStat) {
		return fmt.Errorf("source and destination are the same file")
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
		return fmt.Errorf("destination is inside the source directory: %s", dest)
	}
	return nil
}

func copyFile(src, dest string, force bool) error {
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("destination file '%s' already exists (use -f to overwrite)", dest)
		}
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	size, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer out.Close()

	progress := newProgressBar("Copying "+filepath.Base(src), size.Size())
	if progress != nil {
		defer progress.Finish()
		_, err = io.Copy(io.MultiWriter(out, progress), in)
	} else {
		_, err = io.Copy(out, in)
	}
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
