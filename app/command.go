package main

import (
	"fmt"
	"os"
)

type commandMenu struct {
	commands map[string]command
}

type command interface {
	execute(param string)
}

type exit struct {
	name string
}

func (e exit) execute(param string) {
	if param == "0" {
		os.Exit(0)
	}

	fmt.Println("invalid parameter")
	os.Exit(1)
}

type echo struct {
	name string
}

func (e echo) execute(param string) {
	fmt.Println(param)
}


func newCommandMenu() commandMenu {
	menu := commandMenu{
		commands: map[string]command{
			"exit": exit{
				name: "exit",
			},
			"echo": echo{
				name: "echo",
			},
		},
	}
	return menu
}