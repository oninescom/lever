package main

import (
	"fmt"
	"lever/commands"
	"lever/engine"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var rootCmd = &engine.Command{
		Use:   "lever",
		Short: "lever engine",
		Long:  `一个采用纯 Go 语言精细化压榨体积、无沉重反射损耗、整合了网络与系统调度的全能极客开发者控制台。`,
	}

	rootCmd.AddCommand(
		commands.NewLsCmd(),
		commands.NewCatCmd(),
		commands.NewGrepCmd(),
		commands.NewTouchCmd(),
		commands.NewMkdirCmd(),
		commands.NewCpCmd(),
		commands.NewMvCmd(),
		commands.NewRmCmd(),
		commands.NewPwdCmd(),
		commands.NewClearCmd(),
		commands.NewNcCmd(),
		commands.NewDuCmd(),
		commands.NewAwkCmd(),
	)

	execName := filepath.Base(os.Args[0])
	execName = strings.TrimSuffix(execName, filepath.Ext(execName))

	if execName != "lever" && execName != "main" {
		for _, child := range rootCmd.Children {
			if engine.CommandName(child.Use) == execName {
				if err := child.ExecuteArgs(os.Args[1:]); err != nil {
					fmt.Fprintf(os.Stderr, "lever 运行错误: %v\n", err)
					os.Exit(1)
				}
				return
			}
		}
		fmt.Fprintf(os.Stderr, "lever 错误: 工具箱中未注册名为 '%s' 的专属子命令。\n", execName)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "lever 错误: %v\n", err)
		os.Exit(1)
	}
}
