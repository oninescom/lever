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
		Long:  `A compact, pure Go developer toolbox for file, text, network, and system tasks.`,
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
		commands.NewTailCmd(),
		commands.NewCurlCmd(),
		commands.NewHashCmd(),
		commands.NewPingCmd(),
		commands.NewSedCmd(),
		commands.NewUnameCmd(),
		commands.NewWcCmd(),
		commands.NewHashCmd(),
	)

	rootCmd.AddCommand(
		commands.NewInstallCmd(rootCmd),
		commands.NewUninstallCmd(),
	)

	execName := filepath.Base(os.Args[0])
	execName = strings.TrimSuffix(execName, filepath.Ext(execName))

	if execName != "lever" && execName != "main" {
		for _, child := range rootCmd.Children {
			if engine.CommandName(child.Use) == execName {
				if err := child.ExecuteArgs(os.Args[1:]); err != nil {
					fmt.Fprintf(os.Stderr, "lever execution error: %v\n", err)
					os.Exit(1)
				}
				return
			}
		}
		fmt.Fprintf(os.Stderr, "lever error: no command is registered for '%s'.\n", execName)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "lever error: %v\n", err)
		os.Exit(1)
	}
}
