package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
	"time"
)

func NewTouchCmd() *engine.Command {
	return &engine.Command{
		Use:   "touch [file...]",
		Short: "Linux 风格的 touch 工具",
		Long:  `在 Windows 中快速创建空文件，或者更新已有文件的时间戳（访问时间和修改时间）。`,
		Args:  utils.MinimumNArgs(1),
		Run: func(c *engine.Command, args []string) {
			now := time.Now()
			for _, filename := range args {
				file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
				if err != nil {
					fmt.Fprintf(os.Stderr, "touch 错误: 无法处理文件 %s: %v\n", filename, err)
					continue
				}
				file.Close()

				err = os.Chtimes(filename, now, now)
				if err != nil {
					fmt.Fprintf(os.Stderr, "touch 错误: 无法更新 %s 的时间戳: %v\n", filename, err)
				}
			}
		},
	}
}
