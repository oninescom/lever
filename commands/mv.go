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
		Short: "Move or rename files and directories",
		Long:  `Move or rename files and directories, including multiple sources and moves across volumes.`,
		Args:  utils.MinimumNArgs(2),
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			sources := flags.Args()[:len(flags.Args())-1]
			dest := flags.Args()[len(flags.Args())-1]

			if len(sources) > 1 && !utils.ValidMultiSourceDestination(sources, dest) {
				return fmt.Errorf("mv error: destination must be an existing directory when moving multiple files")
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
					failures = append(failures, fmt.Errorf("mv error: cannot move '%s': %w", src, err))
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
	// ERROR_NOT_SAME_DEVICE indicates a cross-volume move; other errors must propagate.
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
