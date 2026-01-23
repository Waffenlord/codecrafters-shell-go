package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type builtin func(in io.Reader, out io.Writer, args []string, termState *term.State) error

type builtInMenu struct {
	commands   map[string]builtin
	prefixTrie *trieNode
}

func (bM builtInMenu) isBuiltIn(cmd string) bool {
	_, ok := bM.commands[cmd]
	return ok
}

var typeCmd builtin

var builtInCommandMap = map[string]builtin{
	"exit": exit,
	"echo": echo,
	"type": typeCmd,
	"pwd":  pwd,
	"cd":   cd,
	"history": history,
}

func init() {
	typeCmd = func(_ io.Reader, out io.Writer, args []string, _ *term.State) error {
		cmd := strings.Join(args, "")
		_, ok := builtInCommandMap[cmd]
		if !ok {
			path := getCommandDirectoryAsync(cmd)
			if path != "" {
				fmt.Fprintf(out, "%s is %s\n", cmd, path)
				return nil
			}
			fmt.Fprintf(out, "%s: not found\n", cmd)
			return nil
		}
		fmt.Fprintf(out, "%s is a shell builtin\n", cmd)
		return nil
	}
	builtInCommandMap["type"] = typeCmd
}

func newBuiltInMenu() builtInMenu {
	return builtInMenu{
		commands:   builtInCommandMap,
		prefixTrie: getCommandsTrie(builtInCommandMap),
	}
}

func exit(_ io.Reader, _ io.Writer, args []string, termState *term.State) error {
	fmt.Printf("\r\n")
	term.Restore(int(os.Stdin.Fd()), termState)
	os.Exit(0)

	return nil
}

func echo(_ io.Reader, out io.Writer, args []string, _ *term.State) error {
	fmt.Fprintln(out, strings.Trim(strings.Join(args, ""), " "))
	return nil
}

func pwd(_ io.Reader, out io.Writer, args []string, _ *term.State) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return errors.New("error finding path")
	}
	fmt.Fprintln(out, currentDir)
	return nil
}

func cd(_ io.Reader, out io.Writer, args []string, termState *term.State) error {
	path := strings.Join(args, "")
	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			term.Restore(int(os.Stdin.Fd()), termState)
			return errors.New("error finding home directory")
		}
		err = os.Chdir(homeDir)
		if err != nil {
			term.Restore(int(os.Stdin.Fd()), termState)
			return errors.New("error changing to home directory")
		}
		return nil
	}
	_, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(out, "cd: %s: No such file or directory\n", path)
		return nil
	}
	err = os.Chdir(path)
	if err != nil {
		term.Restore(int(os.Stdin.Fd()), termState)
		return errors.New("error changing path")
	}
	return nil
}

func history(_ io.Reader, out io.Writer, args []string, _ *term.State) error {
	return nil
}
