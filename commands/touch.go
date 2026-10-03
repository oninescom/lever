package commands

import (
	"errors"
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
	"time"
)

func NewTouchCmd() *engine.Command {
	return &engine.Command{
		Use:   "touch [file...]",
		Short: "Create files or update timestamps",
		Long:  `Create empty files or update the access and modification times of existing files on Windows.`,
		Args:  utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {
			now := time.Now()
			var failures []error
			for _, filename := range args {
				file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
				if err != nil {
					failures = append(failures, fmt.Errorf("touch error: cannot process %s: %w", filename, err))
					continue
				}
				if err := file.Close(); err != nil {
					failures = append(failures, fmt.Errorf("touch error: cannot close %s: %w", filename, err))
					continue
				}

				err = os.Chtimes(filename, now, now)
				if err != nil {
					failures = append(failures, fmt.Errorf("touch error: cannot update timestamps for %s: %w", filename, err))
				}
			}
			return errors.Join(failures...)
		},
	}
}
