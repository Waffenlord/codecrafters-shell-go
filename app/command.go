package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type commandType string

const (
	builtin commandType = "builtin"
)

type commandMenu struct {
	commands map[string]command
}

type command interface {
	execute(param string) string
	getCategory() commandType
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

func (e exit) execute(param string) string {
	if param == "0" {
		os.Exit(0)
	}

	fmt.Println("invalid parameter")
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

func (e echo) execute(param string) string {
	return param
}

func (e echo) getCategory() commandType {
	return e.category
}

type typeC struct {
	name     string
	category commandType
}

func (t typeC) execute(param string) string {
	c, ok := commandsMap[param]
	if !ok {
		path := getCommandDirectoryAsync(param)
		if path != "" {
			return fmt.Sprintf("%s is %s\n", param, path)
		}
		return fmt.Sprintf("%s: not found\n", param)
	}
	return fmt.Sprintf("%s is a shell %s\n", param, c.getCategory())
}

func (t typeC) getCategory() commandType {
	return t.category
}

type pwd struct {
	name     string
	category commandType
}

func (p pwd) execute(param string) string {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("error finding path")
	}
	return fmt.Sprintln(currentDir)
}

func (p pwd) getCategory() commandType {
	return p.category
}

type cd struct {
	name     string
	category commandType
}

func (c cd) execute(param string) string {
	if param == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("error finding home directory")
		}
		err = os.Chdir(homeDir)
		if err != nil {
			log.Fatal("error changing to home directory")
		}
		return ""
	}
	_, err := os.Stat(param)
	if err != nil {
		fmt.Printf("cd: %s: No such file or directory\n", param)
		return ""
	}
	err = os.Chdir(param)
	if err != nil {
		log.Fatal("error changing path")
	}
	return ""
}

func (c cd) getCategory() commandType {
	return c.category
}

func newCommandMenu() commandMenu {
	menu := commandMenu{
		commands: commandsMap,
	}
	return menu
}

func processBuiltInCommand(c command, params []string) {
	commandParams, destinationSlice, actionT, redirectionT, err := hasOutputRedirection(params)
	if err != nil {
		log.Fatal(err)
	}
	input := strings.Join(commandParams, "")
	output := c.execute(input)
	if output != "" {
		if actionT == redirectFile && redirectionT == successOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := writeContentTofile([]byte(output), destination)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		if actionT == redirectFile && redirectionT == errorOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := writeContentTofile([]byte(""), destination)
			if err != nil {
				log.Fatal(err)
			}
		}

		if actionT == appendFile && redirectionT == successOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := appendContentToFile(output, destination)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		if actionT == appendFile && redirectionT == errorOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := appendContentToFile("", destination)
			if err != nil {
				log.Fatal(err)
			}
		}

		fmt.Print(output)
		if len(output) > 0 && output[len(output)-1] != '\n' {
			fmt.Println()
		}
	}
}
