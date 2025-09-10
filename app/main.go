package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type commandReceived struct {
	command string
	params  []string
}

func main() {
	commandMenu := newCommandMenu()
	for {
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		commandTyped, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatal("error occurred while reading your command")
		}
		commandData := cleanCommand(commandTyped)
		result, ok := commandMenu.commands[commandData.command]
		if !ok {
			path := getCommandDirectoryAsync(commandData.command)
			if path != "" {
				out, err := exec.Command(commandData.command, filterSpacesFromParams(commandData.params)...).Output()
				if err != nil {
					log.Fatal(err, string(out))
				}
				fmt.Print(string(out))
				continue
			}
			fmt.Println(commandTyped[:len(commandTyped)-1] + ": command not found")
			continue
		}
		result.execute(strings.Join(commandData.params, ""))
	}

}

func cleanCommand(c string) commandReceived {
	commandCleaned := c[:len(c)-1]
	if len(commandCleaned) == 0 {
		return commandReceived{}
	}
	command, commandParams := parseInput(commandCleaned)

	return commandReceived{
		command,
		commandParams,
	}
}
