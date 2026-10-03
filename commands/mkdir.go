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
		Short: "Create directories",
		Long:  `Create directories. Use -p to create missing parent directories.`,
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
					failures = append(failures, fmt.Errorf("mkdir error: cannot create directory '%s': %w", dir, err))
				}
			}
			return errors.Join(failures...)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&parents, "parents", "p", false, "Create missing parent directories")
	return cmd
}
