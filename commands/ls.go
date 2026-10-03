package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

func NewLsCmd() *engine.Command {
	var showAll, longFormat, humanReadable bool

	cmd := &engine.Command{
		Use:   "ls [path]",
		Short: "List directory contents",
		Long:  `List directory contents with colors, aligned columns, and human-readable sizes.`,
		Args:  utils.MaximumNArgs(1),
		Run: func(c *engine.Command, args []string) {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			files, err := os.ReadDir(dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ls error: cannot read directory %s: %v\n", dir, err)
				os.Exit(1)
			}

			var w *tabwriter.Writer
			if longFormat {
				w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			}

			for _, f := range files {
				name := f.Name()
				isHidden := (len(name) > 0 && name[0] == '.') || utils.IsWindowsHidden(filepath.Join(dir, name))
				if !showAll && isHidden {
					continue
				}
				colorCode := ""
				if f.IsDir() {
					colorCode = utils.ColorBoldBlue
				} else {
					ext := strings.ToLower(filepath.Ext(name))
					if ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".ps1" {
						colorCode = utils.ColorGreen
					}
				}

				displayName := name
				if colorCode != "" {
					displayName = colorCode + name + utils.ColorReset
				}

				if longFormat {
					info, err := f.Info()
					if err != nil {
						continue
					}

					var sizeStr string
					if humanReadable {
						sizeStr = utils.FormatSize(info.Size())
					} else {
						sizeStr = fmt.Sprintf("%d", info.Size())
					}

					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						info.Mode().String(),
						sizeStr,
						info.ModTime().Format("Jan 02 15:04"),
						displayName,
					)
				} else {
					fmt.Printf("%s  ", displayName)
				}
			}
			if longFormat {
				w.Flush()
			} else {
				fmt.Println()
			}
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&showAll, "all", "a", false, "Show all files, including hidden files")
	flags.BoolVarP(&longFormat, "long", "l", false, "Show details in long format")
	flags.BoolVarP(&humanReadable, "human-readable", "h", false, "Show file sizes in human-readable form (for example, 1K or 234M)")

	return cmd
}
