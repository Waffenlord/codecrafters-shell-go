package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"

	"golang.org/x/term"
)

type commandReceived struct {
	command string
	params  []string
}

const terminalChar = "$ "

func main() {
	commandMenu := newCommandMenu()
	fmt.Fprint(os.Stdout, terminalChar)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var buffer strings.Builder
	input := make([]byte, 1)
	tabCounter := 0
	for {
		n, err := os.Stdin.Read(input)
		if err != nil || n == 0 {
			continue
		}
		key := input[0]

		switch key {
		case 3: // Ctrl+C
			fmt.Print("\r\nExiting.\r\n")
			return

		case '\r', '\n': // ENTER
			commandTyped := buffer.String()
			if len(commandTyped) == 0 {
				fmt.Print("\r\n")
				fmt.Print(terminalChar)
				continue
			}
			commandData := cleanCommand(commandTyped)
			builtInCommand, ok := commandMenu.commands[commandData.command]
			if !ok {
				path := getCommandDirectoryAsync(commandData.command)
				if path != "" {
					paramsWithoutSpaces := filterSpacesFromParams(commandData.params)
					commandParams, destinationSlice, actionT, redirectionT, err := hasOutputRedirection(paramsWithoutSpaces)
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

					cmd.Wait()

					stdout := string(stdoutBytes)
					stderr := string(stderrBytes)

					processExternalCommandOutput(stdout, stdoutBytes, stderr, stderrBytes, destinationSlice, actionT, redirectionT)
					buffer.Reset()
					fmt.Printf("\r\n%s", terminalChar)
					continue
				}
				fmt.Printf("\r\n%s: command not found\r\n", commandTyped)
				buffer.Reset()
				fmt.Print(terminalChar)
				continue
			}
			processBuiltInCommand(builtInCommand, commandData.params, oldState)
			buffer.Reset()
			fmt.Printf("\r\n%s", terminalChar)

		case '\t': // TAB
			current := buffer.String()
			if len(current) >= 3 {
				matches := commandMenu.prefixTrie.prefixSearch(current)
				if len(matches) == 0 {
					fmt.Print("\x07")
					continue
				}
				if len(matches) == 1 {
					currentMatch := matches[0]
					buffer.Reset()
					buffer.WriteString(currentMatch + " ")
					fmt.Print("\033[2K\r")
					fmt.Printf("$ %s ", currentMatch)
				}
				if len(matches) > 1 {
					if tabCounter == 0 {
						fmt.Print("\x07")
						tabCounter += 1
						continue
					} else {
						slices.Sort(matches)
						currentMatch := strings.Join(matches, "  ")
						fmt.Print("\r\n")
						fmt.Print(currentMatch + "\r\n")
						fmt.Printf("$ %s", buffer.String())
						tabCounter = 0
					}
				}
			}

		case 127: // BACKSPACE
			s := buffer.String()
			if len(s) > 0 {
				s = s[:len(s)-1]
				buffer.Reset()
				buffer.WriteString(s)
				fmt.Printf("\r$ %s \b", s) // redraw current buffer
			}

		default:
			if key >= 32 && key <= 126 {
				buffer.WriteByte(key)
				fmt.Printf("%c", key)
			}
		}
	}

}

func cleanCommand(c string) commandReceived {
	command, commandParams := parseInput(c)

	return commandReceived{
		command,
		commandParams,
	}
}
