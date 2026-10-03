package commands

import (
	"fmt"
	"lever/engine"
	"lever/utils"
	"os"
	"runtime"
	"strings"
)

func NewUnameCmd() *engine.Command {
	var all, sysName, nodeName, release, machine, processor bool

	cmd := &engine.Command{
		Use:   "uname",
		Short: "Linux-style uname utility to print target system and machine architecture metrics",
		Args:  utils.NoArgs,
		RunE: func(c *engine.Command, args []string) error {
			// Fallback option: if no flag is specified, default to -s (Print Kernel Name)
			if !all && !sysName && !nodeName && !release && !machine && !processor {
				sysName = true
			}

			hostname, _ := os.Hostname()
			if hostname == "" {
				hostname = "unknown-node"
			}

			var out []string
			if all || sysName {
				out = append(out, runtime.GOOS)
			} // Kernel name (windows)
			if all || nodeName {
				out = append(out, hostname)
			} // Nodename (Computer network name)
			if all || release {
				out = append(out, "10.0")
			} // Core kernel release layout
			if all || machine {
				out = append(out, runtime.GOARCH)
			} // Machine architecture token (amd64/arm64)
			if all || processor {
				out = append(out, os.Getenv("PROCESSOR_IDENTIFIER"))
			} // Native processor footprint

			fmt.Println(strings.Join(out, " "))
			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&all, "all", "a", false, "Print all system core blueprint details")
	flags.BoolVarP(&sysName, "kernel-name", "s", false, "Print the system operating kernel name")
	flags.BoolVarP(&nodeName, "nodename", "n", false, "Print the local network node hostname")
	flags.BoolVarP(&release, "kernel-release", "r", false, "Print the system kernel release version")
	flags.BoolVarP(&machine, "machine", "m", false, "Print the platform hardware architecture machine type")
	flags.BoolVarP(&processor, "processor", "p", false, "Print the active processor hardware identification topologies")
	return cmd
}
