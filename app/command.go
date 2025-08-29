package main

import (
	"fmt"
	"os"
)

type commandType string

const (
	builtin commandType = "builtin"
)

type commandMenu struct {
	commands map[string]command
}

type command interface {
	execute(param string)
	getCategory() commandType
}

var commandsMap = map[string]command{
	"exit": exit{
		name: "exit",
		category: builtin,
	},
	"echo": echo{
		name: "echo",
		category: builtin,
	},
	"type": typeC {
		name: "type",
		category: builtin,
	},
}

type exit struct {
	name     string
	category commandType
}

func (e exit) execute(param string) {
	if param == "0" {
		os.Exit(0)
	}

	fmt.Println("invalid parameter")
	os.Exit(1)
}

func (e exit) getCategory() commandType {
	return e.category
}

type echo struct {
	name string
	category commandType
}

func (e echo) execute(param string) {
	fmt.Println(param)
}

func (e echo) getCategory() commandType {
	return e.category
}

type typeC struct {
	name string
	category commandType
}

func (t typeC) execute(param string) {
	c, ok := commandsMap[param]
	if !ok {
		path := getCommandDirectoryAsync(param)
		if path != "" {
			fmt.Printf("%s is %s\n", param, path)
			return
		}
		fmt.Printf("%s: not found\n", param)
		return
	}
	fmt.Printf("%s is a shell %s\n", param, c.getCategory())
}

func (t typeC) getCategory() commandType {
	return t.category
}

func newCommandMenu() commandMenu {
	menu := commandMenu{
		commands: commandsMap,
	}
	return menu
}
