package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
)

func NewPwdCmd() *engine.Command {
	return &engine.Command{
		Use:   "pwd",
		Short: "Print the current working directory",
		Long:  `Print the absolute path of the current working directory.`,
		Args:  utils.NoArgs,
		Run: func(c *engine.Command, args []string) {
			dir, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "pwd error: cannot get current working directory: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(dir)
		},
	}
}
