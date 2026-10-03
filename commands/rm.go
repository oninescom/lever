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
		Short: "Remove files or directories",
		Long:  `Remove files or directories. Use -r to remove directories recursively and -f to ignore missing paths.`,
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
					failures = append(failures, fmt.Errorf("rm error: cannot remove '%s': no such file or directory", target))
					continue
				}
				if err != nil {
					failures = append(failures, fmt.Errorf("rm error: cannot inspect '%s': %w", target, err))
					continue
				}

				if stat.IsDir() && !recursive {
					failures = append(failures, fmt.Errorf("rm error: cannot remove '%s': is a directory (use -r)", target))
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
					failures = append(failures, fmt.Errorf("rm error: cannot remove %s: %w", target, err))
				}
			}
			return errors.Join(failures...)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&recursive, "recursive", "r", false, "Remove directories and their contents recursively")
	flags.BoolVarP(&force, "force", "f", false, "Ignore missing files and suppress warnings")

	return cmd
}
