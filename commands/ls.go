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
		Short: "Linux 风格的 ls 目录列出工具",
		Long:  `优雅地列出目标目录下的内容。支持彩色辨识系统、表格自动对齐以及人类可读大小折算。`,
		Args:  utils.MaximumNArgs(1),
		Run: func(c *engine.Command, args []string) {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			// 读取目录文件
			files, err := os.ReadDir(dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ls 错误: 无法读取目录 %s: %v\n", dir, err)
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
					colorCode = utils.ColorBoldBlue // 文件夹显示加粗蓝
				} else {
					ext := strings.ToLower(filepath.Ext(name))
					if ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".ps1" {
						colorCode = utils.ColorGreen // 可执行程序/脚本显示绿色
					}
				}

				// 拼接带有 ANSI 颜色控制符的文件名
				displayName := name
				if colorCode != "" {
					displayName = colorCode + name + utils.ColorReset
				}

				// 3. 输出格式化分流
				if longFormat {
					info, err := f.Info()
					if err != nil {
						continue
					}

					// 格式化文件大小
					var sizeStr string
					if humanReadable {
						sizeStr = utils.FormatSize(info.Size())
					} else {
						sizeStr = fmt.Sprintf("%d", info.Size())
					}

					// 使用 \t 分隔字段，tabwriter 会自动计算最大宽度并完美按列对齐
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						info.Mode().String(),
						sizeStr,
						info.ModTime().Format("Jan 02 15:04"),
						displayName,
					)
				} else {
					// 普通模式：横向并排打印
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
	flags.BoolVarP(&showAll, "all", "a", false, "显示所有文件，包括隐藏文件")
	flags.BoolVarP(&longFormat, "long", "l", false, "使用长列表格式显示详细信息")
	flags.BoolVarP(&humanReadable, "human-readable", "h", false, "以人类可读的格式打印文件大小 (例如 1K 234M)")

	return cmd
}
