package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"

	"golang.org/x/term"
)

type commandReceived struct {
	command string
	params  []string
}

const terminalChar = "$ "

func main() {
	commandMenu := newBuiltInMenu()
	fmt.Fprint(os.Stdout, terminalChar)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var buffer strings.Builder
	input := make([]byte, 3)
	tabCounter := 0
	for {
		n, err := os.Stdin.Read(input)
		if err != nil || n == 0 {
			continue
		}

		for i := 0; i < n; i++ {
			key := input[i]

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
				commandMenu.history = append(commandMenu.history, commandTyped)
				commandMenu.cmdIndex = len(commandMenu.history)
				commands, hasPipeline := parseInput(commandTyped)
				if len(commands) == 0 {
					fmt.Printf("\r\n%s: command not found\r\n", commandTyped)
					buffer.Reset()
					fmt.Print(terminalChar)
					continue
				}

				if hasPipeline {
					fmt.Print("\r\n")
					term.Restore(int(os.Stdin.Fd()), oldState)
					err := processPipeline(commands, commandMenu, oldState)
					if err != nil {
						log.Fatal(err)
					}
					oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
					if err != nil {
						panic(err)
					}
					buffer.Reset()
					fmt.Printf("\r%s", terminalChar)
					continue
				}

				commandData := commands[0]
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

						var stdoutBuf, stderrBuf bytes.Buffer

						stdoutPipe, err := cmd.StdoutPipe()
						if err != nil {
							log.Fatal(err)
						}

						stderrPipe, err := cmd.StderrPipe()
						if err != nil {
							log.Fatal(err)
						}

						if err := cmd.Start(); err != nil {
							log.Fatal(err)
						}

						stdoutWriter := io.Writer(&stdoutBuf)
						stderrWriter := io.Writer(&stderrBuf)

						var wg sync.WaitGroup
						wg.Add(2)

						go func() {
							defer wg.Done()
							io.Copy(stdoutWriter, stdoutPipe)
						}()

						go func() {
							defer wg.Done()
							io.Copy(stderrWriter, stderrPipe)
						}()

						cmd.Wait()
						wg.Wait()

						stdoutBytes := stdoutBuf.Bytes()
						stderrBytes := stderrBuf.Bytes()

						stdout := stdoutBuf.String()
						stderr := stderrBuf.String()

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

				buffer.Reset()
				var output bytes.Buffer
				commandParams, destinationSlice, actionT, redirectionT, err := hasOutputRedirection(commandData.params)
				if err != nil {
					term.Restore(int(os.Stdin.Fd()), oldState)
					log.Fatal(err)
				}
				err = builtInCommand(os.Stdin, &output, commandParams, oldState, &commandMenu.history)
				if err != nil {
					term.Restore(int(os.Stdin.Fd()), oldState)
					log.Fatal(err)
				}
				shouldPrint, err := checkRedirection(output, destinationSlice, actionT, redirectionT, oldState)
				if err != nil {
					term.Restore(int(os.Stdin.Fd()), oldState)
					log.Fatal(err)
				}
				if shouldPrint && output.Len() > 0 {
					fmt.Printf("\r\n%s", output.String())
					fmt.Printf("\r%s", terminalChar)
				} else {
					fmt.Printf("\r\n%s", terminalChar)
				}

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
						commonPrefix := findLongestCommonPrefix(matches)
						if len(commonPrefix) > 0 && buffer.String() != commonPrefix {
							buffer.Reset()
							buffer.WriteString(commonPrefix)
							fmt.Print("\033[2K\r")
							fmt.Printf("$ %s", commonPrefix)
							continue
						}
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

			case 27:
				if i+2 < n && input[i+1] == '[' {
					if len(commandMenu.history) == 0 {
						buffer.Reset()
						fmt.Print("\033[2K\r")
						fmt.Print(terminalChar)
						i += 2
						continue
					}
					switch input[i+2] {
					case 'A': // UP ARROW
						if commandMenu.cmdIndex > 0 {
							commandMenu.cmdIndex -= 1
						}
						cmd := commandMenu.history[commandMenu.cmdIndex]
						buffer.Reset()
						buffer.WriteString(cmd)
						fmt.Print("\033[2K\r")
						fmt.Printf("$ %s", cmd)

					case 'B': // DOWN ARROW
						if commandMenu.cmdIndex < len(commandMenu.history) - 1 {
							commandMenu.cmdIndex += 1
						}
						cmd := commandMenu.history[commandMenu.cmdIndex]
						buffer.Reset()
						buffer.WriteString(cmd)
						fmt.Print("\033[2K\r")
						fmt.Printf("$ %s", cmd)
					}
					i += 2
				}

			default:
				if key >= 32 && key <= 126 {
					buffer.WriteByte(key)
					fmt.Printf("%c", key)
				}
			}
		}
	}

}
