package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"
)

type commandType string

const (
	builtin commandType = "builtin"
)

type commandMenu struct {
	commands   map[string]command
	prefixTrie *trieNode
}

type command interface {
	execute(execArgs) string
	getCategory() commandType
}

type execArgs struct {
	param        string
	termOldState *term.State
}

var commandsMap = map[string]command{
	"exit": exit{
		name:     "exit",
		category: builtin,
	},
	"echo": echo{
		name:     "echo",
		category: builtin,
	},
	"type": typeC{
		name:     "type",
		category: builtin,
	},
	"pwd": pwd{
		name:     "pwd",
		category: builtin,
	},
	"cd": cd{
		name:     "cd",
		category: builtin,
	},
}

type exit struct {
	name     string
	category commandType
}

// Need to restore the terminal
func (e exit) execute(commandArgs execArgs) string {
	if commandArgs.param == "0" {
		fmt.Printf("\r\n")
		term.Restore(int(os.Stdin.Fd()), commandArgs.termOldState)
		os.Exit(0)
	}

	fmt.Printf("\r\ninvalid parameter\r\n")
	term.Restore(int(os.Stdin.Fd()), commandArgs.termOldState)
	os.Exit(1)
	return ""
}

func (e exit) getCategory() commandType {
	return e.category
}

type echo struct {
	name     string
	category commandType
}

func (e echo) execute(commandArgs execArgs) string {
	return commandArgs.param
}

func (e echo) getCategory() commandType {
	return e.category
}

type typeC struct {
	name     string
	category commandType
}

func (t typeC) execute(commandArgs execArgs) string {
	c, ok := commandsMap[commandArgs.param]
	if !ok {
		path := getCommandDirectoryAsync(commandArgs.param)
		if path != "" {
			return fmt.Sprintf("%s is %s", commandArgs.param, path)
		}
		return fmt.Sprintf("%s: not found", commandArgs.param)
	}
	return fmt.Sprintf("%s is a shell %s", commandArgs.param, c.getCategory())
}

func (t typeC) getCategory() commandType {
	return t.category
}

type pwd struct {
	name     string
	category commandType
}

func (p pwd) execute(commandArgs execArgs) string {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("error finding path")
	}
	return currentDir
}

func (p pwd) getCategory() commandType {
	return p.category
}

type cd struct {
	name     string
	category commandType
}

func (c cd) execute(commandArgs execArgs) string {
	if commandArgs.param == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			term.Restore(int(os.Stdin.Fd()), commandArgs.termOldState)
			log.Fatal("error finding home directory")
		}
		err = os.Chdir(homeDir)
		if err != nil {
			term.Restore(int(os.Stdin.Fd()), commandArgs.termOldState)
			log.Fatal("error changing to home directory")
		}
		return ""
	}
	_, err := os.Stat(commandArgs.param)
	if err != nil {
		fmt.Printf("\r\ncd: %s: No such file or directory", commandArgs.param)
		return ""
	}
	err = os.Chdir(commandArgs.param)
	if err != nil {
		term.Restore(int(os.Stdin.Fd()), commandArgs.termOldState)
		log.Fatal("error changing path")
	}
	return ""
}

func (c cd) getCategory() commandType {
	return c.category
}

func newCommandMenu() commandMenu {
	menu := commandMenu{
		commands:   commandsMap,
		prefixTrie: getCommandsTrie(commandsMap),
	}
	return menu
}

func processBuiltInCommand(c command, params []string, termOldState *term.State) {
	commandParams, destinationSlice, actionT, redirectionT, err := hasOutputRedirection(params)
	if err != nil {
		log.Fatal(err)
	}
	input := strings.Join(commandParams, "")
	output := c.execute(execArgs{param: input, termOldState: termOldState})
	if output != "" {
		if actionT == redirectFile && redirectionT == successOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := writeContentTofile([]byte(output), destination)
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), termOldState)
				log.Fatal(err)
			}
			return
		}
		if actionT == redirectFile && redirectionT == errorOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := writeContentTofile([]byte(""), destination)
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), termOldState)
				log.Fatal(err)
			}
		}

		if actionT == appendFile && redirectionT == successOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := appendContentToFile(output, destination)
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), termOldState)
				log.Fatal(err)
			}
			return
		}

		if actionT == appendFile && redirectionT == errorOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := appendContentToFile("", destination)
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), termOldState)
				log.Fatal(err)
			}
		}

		fmt.Printf("\r\n%s", output)
	}
}

func getCommandsTrie(builtin map[string]command) *trieNode {
	t := createTrie()

	// Insert builtin commands
	for k := range builtin {
		t.insert(k)
	}

	return t
}
