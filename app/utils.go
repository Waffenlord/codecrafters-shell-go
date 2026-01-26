package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

func getPathDirectories() []string {
	pathValue := os.Getenv("PATH")
	directories := strings.Split(pathValue, ":")
	return directories
}

func getFileNamesFromDirectories() []string {
	dirs := getPathDirectories()
	filenames := []string{}

	for _, path := range dirs {
		entries, err := os.ReadDir(path)
		if err != nil {
			continue
		}
		for _, entrie := range entries {
			if !entrie.IsDir() {
				filenames = append(filenames, entrie.Name())
			}
		}

	}
	return filenames
}

func isFileExecutable(info os.FileInfo) bool {
	mode := info.Mode()
	return mode.IsRegular() && (mode&0111 != 0) // any executable bit set
}

func getCommandsTrie(builtString map[string]builtin) *trieNode {
	t := createTrie()

	// Insert builtString commands
	for k := range builtString {
		t.insert(k)
	}

	pathFiles := getFileNamesFromDirectories()
	for _, file := range pathFiles {
		t.insert(file)
	}

	return t
}

/*
func getCommandDirectory(c string) string {
	directories := getPathDirectories()
	for _, dir := range directories {
		currentPath := strings.Join([]string{dir, c}, "/")
		info, err := os.Stat(currentPath)
		if err != nil {
			continue
		}
		if isFileExecutable(info) {
			return currentPath
		}
	}

	return ""
}
*/

// Async implementation
func getCommandDirectoryAsync(c string) string {
	result := make(chan string)
	directories := getPathDirectories()

	var once sync.Once
	var wg sync.WaitGroup

	wg.Add(len(directories))

	for _, dir := range directories {
		currentPath := strings.Join([]string{dir, c}, "/")
		go func(path string) {
			defer wg.Done()
			info, err := os.Stat(path)
			if err == nil && isFileExecutable(info) {
				// Only the first successful goroutine should send/close
				once.Do(func() {
					result <- path
					close(result)
				})
			}
		}(currentPath)
	}

	go func() {
		wg.Wait()
		once.Do(func() {
			close(result)
		})
	}()

	for path := range result {
		return path
	}

	return ""
}

func filterSpacesFromParams(params []string) []string {
	filtered := []string{}
	var currentArg string
	for _, p := range params {
		if p == " " {
			filtered = append(filtered, currentArg)
			currentArg = ""
			continue
		}
		currentArg += p
	}
	if currentArg != "" {
		filtered = append(filtered, currentArg)
	}
	return filtered
}

type redirectionType string

const (
	successOut redirectionType = "successOut"
	errorOut   redirectionType = "errorOut"
)

type actionType string

const (
	redirectFile actionType = "redirect"
	appendFile   actionType = "append"
)

func hasOutputRedirection(params []string) ([]string, []string, actionType, redirectionType, error) {
	var commandParams []string
	var destination []string

	for i, p := range params {
		if p == ">" || p == "1>" || p == "2>" {
			if i+1 >= len(params) {
				return nil, nil, "", "", errors.New("invalid destination")
			}
			commandParams = params[:i]
			destination = params[i+1:]
			var redirType redirectionType
			switch p {
			case "2>":
				redirType = errorOut
			default:
				redirType = successOut
			}
			return commandParams, destination, redirectFile, redirType, nil
		}
		if p == ">>" || p == "1>>" || p == "2>>" {
			if i+1 >= len(params) {
				return nil, nil, "", "", errors.New("invalid destination")
			}
			commandParams = params[:i]
			destination = params[i+1:]
			var redirType redirectionType
			switch p {
			case "2>>":
				redirType = errorOut
			default:
				redirType = successOut
			}
			return commandParams, destination, appendFile, redirType, nil
		}
	}
	return params, nil, "", "", nil
}

func writeContentTofile(content []byte, destination string) error {
	err := os.WriteFile(destination, content, 0644)
	if err != nil {
		return err
	}
	return nil
}

func appendContentToFile(content string, destination string) error {
	f, err := os.OpenFile(destination, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return err
	}
	return nil
}

func checkRedirection(output bytes.Buffer, destinationSlice []string, actionT actionType, redirectionT redirectionType, termOldState *term.State) (bool, error) {
	if output.Len() > 0 {
		if actionT == redirectFile && redirectionT == successOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := writeContentTofile([]byte(output.Bytes()), destination)
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), termOldState)
				return false, err
			}
			return false, nil
		}
		if actionT == redirectFile && redirectionT == errorOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := writeContentTofile([]byte(""), destination)
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), termOldState)
				return false, err
			}
			return true, nil
		}

		if actionT == appendFile && redirectionT == successOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := appendContentToFile(output.String(), destination)
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), termOldState)
				return false, err
			}
			return false, nil
		}

		if actionT == appendFile && redirectionT == errorOut {
			destination := strings.Trim(strings.Join(destinationSlice, ""), " ")
			err := appendContentToFile("", destination)
			if err != nil {
				term.Restore(int(os.Stdin.Fd()), termOldState)
				return false, err
			}
			return true, nil
		}

		return true, nil
	}
	return false, nil
}

func processExternalCommandOutput(
	successString string,
	successBytes []byte,
	errorString string,
	errorBytes []byte,
	destinationSlice []string,
	actionT actionType,
	rT redirectionType,
) {
	if actionT == redirectFile && rT == successOut {
		destination := destinationSlice[0]
		err := writeContentTofile([]byte(transformNewLines(successString)), destination)
		if err != nil {
			log.Fatal(err)
		}
		if len(errorString) > 0 {
			fmt.Printf("\r\n%s", removeNewLines(errorString))
		}
		return
	}

	if actionT == redirectFile && rT == errorOut {
		destination := destinationSlice[0]
		err := writeContentTofile([]byte(transformNewLines(errorString)), destination)
		if err != nil {
			log.Fatal(err)
		}
	}

	if actionT == appendFile && rT == successOut {
		destination := destinationSlice[0]
		err := appendContentToFile(successString, destination)
		if err != nil {
			log.Fatal(err)
		}
		if len(errorString) > 0 {
			fmt.Printf("\r\n%s", removeNewLines(errorString))
		}
		return
	}

	if actionT == appendFile && rT == errorOut {
		destination := destinationSlice[0]
		err := appendContentToFile(errorString, destination)
		if err != nil {
			log.Fatal(err)
		}
	}
	if len(successString) > 0 {
		fmt.Printf("\r\n%s", transformNewLines(successString))
	}
	if len(errorString) > 0 && actionT == "" && rT != errorOut {
		fmt.Println("Executing error")
		fmt.Printf("\r%s\r\n", errorString)
	}

}

func removeNewLines(content string) string {
	if len(content) > 0 && content[len(content)-1] == '\n' {
		return content[:len(content)-1]
	}
	return content
}

func transformNewLines(content string) string {
	return removeNewLines(strings.ReplaceAll(content, "\n", "\r\n"))
}

type historyRecord struct {
	order int
	value string
}

func parseHistoryList(history []string) []historyRecord {
	result := []historyRecord{}
	for i, val := range history {
		result = append(result, historyRecord{
			order: i + 1,
			value: val,
		})
	}

	return result
}

func processHistoryLimit(history []historyRecord, limit int) []historyRecord {
	if limit >= len(history) {
		return history
	}
	startIndex := len(history) - limit
	return history[startIndex:]
}