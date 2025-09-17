package main

import (
	"errors"
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

	go func(){
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

func hasOutputRedirection(params []string) ([]string, []string, bool, error) {
	var commandParams []string
	var destination []string

	for i, p := range params {
		if p == ">" || p == "1>" {
			if i + 1 >= len(params) {
				return nil, nil, false, errors.New("invalid destination")
			}
			commandParams = params[:i]
			destination = params[i+1:]
			return commandParams, destination, true, nil 
		}
	}
	return params, nil, false, nil
}

func writeContentTofile(content []byte, destination string) error {
	err := os.WriteFile(destination, content, 0644)
	if err != nil {
		return err
	}
	return nil
}


