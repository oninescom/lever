package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
)

func NewClearCmd() *engine.Command {
	return &engine.Command{
		Use:   "clear",
		Short: "Clear the terminal screen",
		Long:  `Clear the terminal screen and move the cursor to the top left.`,
		Args:  utils.NoArgs,
		Run: func(c *engine.Command, args []string) {
			fmt.Print("\033[H\033[2J")
		},
	}
}
