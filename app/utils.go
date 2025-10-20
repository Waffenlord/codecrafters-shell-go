package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

func getPathDirectories() []string {
	pathValue := os.Getenv("PATH")
	directories := strings.Split(pathValue, ":")
	return directories
}

func isFileExecutable(info os.FileInfo) bool {
	mode := info.Mode()
	return mode.IsRegular() && (mode&0111 != 0) // any executable bit set
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
	appendFile actionType = "append"
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
	fileInfo, err := os.Stat(destination)
	hasContent := false
	if err == nil && fileInfo.Size() > 0 {
		hasContent = true
	}

	f, err := os.OpenFile(destination, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if hasContent {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	cleanContent := content
	if len(content) > 0 && content[len(content) - 1] == '\n' {
		cleanContent = content[:len(content) - 1]
	}


	if _, err := f.WriteString(cleanContent); err != nil {
		return err
	}
	return nil
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
		err := writeContentTofile(successBytes, destination)
		if err != nil {
			log.Fatal(err)
		}
		if len(errorString) > 0 {
			fmt.Print(errorString)
			if len(errorString) > 0 && errorString[len(errorString)-1] != '\n' {
				fmt.Println()
			}
		}
		return
	}

	if actionT == redirectFile && rT == errorOut {
		destination := destinationSlice[0]
		err := writeContentTofile(errorBytes, destination)
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
			fmt.Print(errorString)
			if len(errorString) > 0 && errorString[len(errorString)-1] != '\n' {
				fmt.Println()
			}
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

	fmt.Print(successString)
	if len(successString) > 0 && successString[len(successString)-1] != '\n' {
		fmt.Println()
	}
	if len(errorString) > 0 && actionT == "" && rT != errorOut {
		fmt.Print(errorString)
		if len(errorString) > 0 && errorString[len(errorString)-1] != '\n' {
			fmt.Println()
		}
	}

}
