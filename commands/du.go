package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"lever/engine"
	"lever/utils"
	"os"
	"path/filepath"
)

type diskUsage struct {
	path string
	size int64
}

func NewDuCmd() *engine.Command {
	var summarize bool
	var humanReadable bool

	cmd := &engine.Command{
		Use:   "du [path...]",
		Short: "Show disk usage",
		Long:  `Show disk usage for files or directories. Use -s for totals and -h for human-readable sizes.`,
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			targets := flags.Args()

			if len(targets) == 0 {
				targets = []string{"."}
			}
			var failures []error
			for _, target := range targets {
				entries, err := scanDiskUsage(target, summarize)
				if err != nil {
					failures = append(failures, fmt.Errorf("du error: cannot read '%s': %w", target, err))
					continue
				}
				for _, entry := range entries {
					printSize(entry.size, entry.path, humanReadable)
				}
			}
			return errors.Join(failures...)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&summarize, "summarize", "s", false, "Show only the total for each argument")
	flags.BoolVarP(&humanReadable, "human-readable", "h", false, "Show sizes in human-readable form (for example, 1K or 234M)")

	return cmd
}

func scanDiskUsage(path string, summarize bool) ([]diskUsage, error) {
	path = filepath.Clean(path)
	stat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		if summarize {
			return []diskUsage{{path: path, size: stat.Size()}}, nil
		}
		return nil, nil
	}

	var entries []diskUsage
	var total int64
	sizes := make(map[string]int64)
	err = filepath.WalkDir(path, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return duWalkError(path, current, err)
		}
		if entry.IsDir() {
			if !summarize {
				entries = append(entries, diskUsage{path: current})
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return duWalkError(path, current, err)
		}
		total += info.Size()
		if !summarize {
			sizes[filepath.Dir(current)] += info.Size()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if summarize {
		return []diskUsage{{path: path, size: total}}, nil
	}
	for i := len(entries) - 1; i >= 0; i-- {
		entries[i].size = sizes[entries[i].path]
		sizes[filepath.Dir(entries[i].path)] += entries[i].size
	}
	return entries, nil
}

func duWalkError(root, current string, err error) error {
	if current != root && errors.Is(err, fs.ErrPermission) {
		return nil
	}
	return err
}

func printSize(size int64, path string, human bool) {
	sizeStr := fmt.Sprintf("%d", size)
	if human {
		sizeStr = utils.FormatSize(size)
	}
	fmt.Printf("%-10s %s\n", sizeStr, path)
}
