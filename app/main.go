package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type commandReceived struct {
	command string
	param string 
}

func main() {
	commandMenu := newCommandMenu()
	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		commandTyped, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Println("error occurred while reading your command")
			os.Exit(1)
		}
		commandData := cleanCommand(commandTyped)
		result, ok := commandMenu.commands[commandData.command]
		if !ok {
			fmt.Println(commandTyped[:len(commandTyped)-1] + ": command not found")
			continue
		}
		result.execute(commandData.param)
	}

}


func cleanCommand(c string) commandReceived {
	commandCleaned := c[:len(c)-1]
	if len(commandCleaned) == 0 {
		return commandReceived{}
	}
	commandParts := strings.Split(commandCleaned, " ")
	command := strings.Trim(commandParts[0], " ")
	var commandParam string
	if len(commandParts) > 1 {
		commandParam = strings.Trim(strings.Join(commandParts[1:], " "), " ")
	} 
	return commandReceived{
		command,
		commandParam,
	}
}
