package engine

import (
	"fmt"
	"io"
	"lever/utils"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

type Command struct {
	Use, Short, Long string
	Args             func([]string) error
	Run              func(*Command, []string)
	RunE             func(*Command, []string) error
	PrepareArgs      func([]string) []string
	FlagsSet         *pflag.FlagSet
	Children         []*Command
	Parent           *Command
}

func (c *Command) Flags() *pflag.FlagSet {
	if c.FlagsSet == nil {
		c.FlagsSet = pflag.NewFlagSet(CommandName(c.Use), pflag.ContinueOnError)
		c.FlagsSet.SetOutput(io.Discard)
	}
	return c.FlagsSet
}

func (c *Command) AddCommand(children ...*Command) {
	for _, child := range children {
		child.Parent = c
		c.Children = append(c.Children, child)
	}
}

func (c *Command) Execute() error { return c.ExecuteArgs(os.Args[1:]) }

func (c *Command) ExecuteArgs(args []string) error {
	if len(c.Children) > 0 {
		if len(args) > 0 && args[0] == "--" {
			return c.Help()
		}
		flags := c.Flags()
		utils.EnsureHelpFlag(flags) // 💡 调用跨包大写函数
		flags.SetInterspersed(false)
		if err := flags.Parse(args); err != nil {
			return err
		}
		helpRequested, err := flags.GetBool("help")
		if err != nil {
			return err
		}
		args = flags.Args()
		if len(args) == 0 {
			return c.Help()
		}
		if args[0] == "help" {
			if len(args) == 1 {
				return c.Help()
			}
			for _, child := range c.Children {
				if CommandName(child.Use) == args[1] {
					return child.Help()
				}
			}
			return fmt.Errorf("未知命令: %s", args[1])
		}
		for _, child := range c.Children {
			if CommandName(child.Use) == args[0] {
				if helpRequested {
					return child.Help()
				}
				return child.ExecuteArgs(args[1:])
			}
		}
		return fmt.Errorf("未知命令: %s", args[0])
	}
	if c.PrepareArgs != nil {
		args = c.PrepareArgs(args)
	}
	positionals, help, err := c.parseFlags(args)
	if err != nil {
		return err
	}
	if help {
		return c.Help()
	}
	if c.Args != nil {
		if err := c.Args(positionals); err != nil {
			return err
		}
	}
	if c.RunE != nil {
		return c.RunE(c, positionals)
	}
	if c.Run != nil {
		c.Run(c, positionals)
	}
	return nil
}

func (c *Command) Help() error {
	description := c.Long
	if description == "" {
		description = c.Short
	}
	if description != "" {
		fmt.Fprintln(os.Stdout, description)
		fmt.Fprintln(os.Stdout)
	}
	usage := c.Use
	if c.Parent != nil {
		usage = c.Parent.Use + " " + usage
	}
	if len(c.Children) > 0 {
		fmt.Fprintf(os.Stdout, "Usage:\n  %s [command]\n\nAvailable Commands:\n", usage)
		for _, child := range c.Children {
			fmt.Fprintf(os.Stdout, "  %-12s %s\n", CommandName(child.Use), child.Short)
		}
		fmt.Fprintln(os.Stdout, "\nUse \"lever [command] --help\" for more information.")
		return nil
	}
	utils.EnsureHelpFlag(c.Flags()) // 💡 调用跨包大写函数
	fmt.Fprintf(os.Stdout, "Usage:\n  %s [flags]\n\nFlags:\n", usage)
	c.FlagsSet.VisitAll(func(flag *pflag.Flag) {
		if flag.Shorthand != "" {
			fmt.Fprintf(os.Stdout, "  -%s, --%-18s %s\n", flag.Shorthand, flag.Name, flag.Usage)
		} else {
			fmt.Fprintf(os.Stdout, "      --%-18s %s\n", flag.Name, flag.Usage)
		}
	})
	return nil
}

func (c *Command) parseFlags(args []string) ([]string, bool, error) {
	flags := c.Flags()
	utils.EnsureHelpFlag(flags) // 💡 调用跨包大写函数
	if err := flags.Parse(args); err != nil {
		return nil, false, err
	}
	help, err := flags.GetBool("help")
	if err != nil {
		return nil, false, err
	}
	return flags.Args(), help, nil
}

func CommandName(usage string) string {
	name, _, _ := strings.Cut(usage, " ")
	return name
}
