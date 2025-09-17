package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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
		builtInCommand, ok := commandMenu.commands[commandData.command]
		if !ok {
			path := getCommandDirectoryAsync(commandData.command)
			if path != "" {
				paramsWithoutSpaces := filterSpacesFromParams(commandData.params)
				commandParams, destinationSlice, hasRedirection, err := hasOutputRedirection(paramsWithoutSpaces)
				if err != nil {
					log.Fatal(err)
				}
				cmd := exec.Command(commandData.command, commandParams...)

				stdoutPipe, _ := cmd.StdoutPipe()
				stderrPipe, _ := cmd.StderrPipe()

				if err := cmd.Start(); err != nil {
					log.Fatal(err)
				}

				stdoutBytes, _ := io.ReadAll(stdoutPipe)
				stderrBytes, _ := io.ReadAll(stderrPipe)

				err = cmd.Wait()

				stdout := string(stdoutBytes)
				stderr := string(stderrBytes)

				if err != nil {
					fmt.Print(stderr)
				}
				if hasRedirection {
					destination := destinationSlice[0]
					err := writeContentTofile(stdoutBytes, destination)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					fmt.Print(stdout)
					if stdout[len(stdout)-1] != '\n' {
						fmt.Println()
					}
				}

				continue
			}
			fmt.Println(commandTyped[:len(commandTyped)-1] + ": command not found")
			continue
		}
		processBuiltInCommand(builtInCommand, commandData.params)
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
