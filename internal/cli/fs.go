package cli

import (
	"fmt"
	"os"
)

func GetFilePathFromArgs(args []string) (filePath string, err error) {
	isDefined := false
	if len(args) > 0 {
		filePath = args[0]
		isDefined = true
	}
	if !isDefined {
		return "", fmt.Errorf("failed to receive any arguments")
	}
	fi, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to check for existence of file at path[%s]: %s", filePath, err)
	}
	if fi.IsDir() {
		return "", fmt.Errorf("failed to get a file at path[%s]: got a directory", filePath)
	}
	return filePath, nil
}
