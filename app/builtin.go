package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

type builtin func(in io.Reader, out io.Writer, args []string, termState *term.State, historyList *[]string) error

type builtInMenu struct {
	commands   map[string]builtin
	prefixTrie *trieNode
	history    []string
	cmdIndex   int
}

func (bM builtInMenu) isBuiltIn(cmd string) bool {
	_, ok := bM.commands[cmd]
	return ok
}

var typeCmd builtin

var builtInCommandMap = map[string]builtin{
	"exit":    exit,
	"echo":    echo,
	"type":    typeCmd,
	"pwd":     pwd,
	"cd":      cd,
	"history": history,
}

func init() {
	typeCmd = func(_ io.Reader, out io.Writer, args []string, _ *term.State, _ *[]string) error {
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

func newBuiltInMenu() *builtInMenu {
	return &builtInMenu{
		commands:   builtInCommandMap,
		prefixTrie: getCommandsTrie(builtInCommandMap),
		history:    []string{},
	}
}

func exit(_ io.Reader, _ io.Writer, args []string, termState *term.State, _ *[]string) error {
	fmt.Printf("\r\n")
	term.Restore(int(os.Stdin.Fd()), termState)
	os.Exit(0)

	return nil
}

func echo(_ io.Reader, out io.Writer, args []string, _ *term.State, _ *[]string) error {
	fmt.Fprintln(out, strings.Trim(strings.Join(args, ""), " "))
	return nil
}

func pwd(_ io.Reader, out io.Writer, args []string, _ *term.State, _ *[]string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return errors.New("error finding path")
	}
	fmt.Fprintln(out, currentDir)
	return nil
}

func cd(_ io.Reader, out io.Writer, args []string, termState *term.State, _ *[]string) error {
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

func history(_ io.Reader, out io.Writer, args []string, _ *term.State, hList *[]string) error {
	var historyOutput string
	existingHistory := *hList
	totalArgs := len(args)
	if totalArgs > 0 {
		first := args[0]
		switch first {
		case "-r":
			if totalArgs < 1 {
				return errors.New("path is required")
			}
			cleanedParams := filterSpacesFromParams(args)
			path := cleanedParams[1]
			var buffer bytes.Buffer
			err := readContentFromFile(&buffer, path)
			if err != nil {
				return err
			}
			entries := strings.Split(buffer.String(), "\n")
			for _, e := range entries {
				if len(e) > 0 {
					*hList = append(*hList, e)
				}
			}
			return nil
		case "-w":
			if totalArgs < 1 {
				return errors.New("path is required")
			}
			cleanedParams := filterSpacesFromParams(args)
			path := cleanedParams[1]
			for _, e := range existingHistory {
				historyOutput += fmt.Sprintf("%s\n", e)
			}
			err := writeContentTofile([]byte(historyOutput), path)
			if err != nil {
				return err
			}
			return nil
		case "-a":
			if totalArgs < 1 {
				return errors.New("path is required")
			}
			cleanedParams := filterSpacesFromParams(args)
			path := cleanedParams[1]
			hasAppended := false
			appendedIndex := 0
			for i, e := range existingHistory {
				if strings.Contains(e, "history -a") && i != len(existingHistory) - 1 {
					hasAppended = true
					appendedIndex = i 
				}
			}

			startIndex := 0
			if hasAppended && appendedIndex + 1 < len(existingHistory) {
				startIndex = appendedIndex + 1
			}

			for startIndex < len(existingHistory) {
				entrie := existingHistory[startIndex]
				historyOutput += fmt.Sprintf("%s\n", entrie)
				startIndex++
			}
			err := appendContentToFile(historyOutput, path)
			if err != nil {
				return err
			}
			return nil
		}
	}

	if len(existingHistory) > 0 {
		currentHistory := parseHistoryList(existingHistory)
		if len(args) > 0 {
			first := args[0]
			n, err := strconv.Atoi(first)
			if err == nil {
				currentHistory = processHistoryLimit(currentHistory, n)
			}
		}

		for _, row := range currentHistory {
			historyOutput += fmt.Sprintf("\t%d %s\r\n", row.order, row.value)
		}
		fmt.Fprint(out, historyOutput)
	}
	return nil
}
