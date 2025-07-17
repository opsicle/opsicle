package cli

import (
	"fmt"
	"os"
)

func GetFilePathFromArgs(args []string) (isDefined bool, filePath string, err error) {
	if len(args) > 0 {
		filePath = args[0]
		isDefined = true
	}
	if !isDefined {
		return false, "", nil
	}
	fi, err := os.Stat(filePath)
	if err != nil {
		return false, "", fmt.Errorf("failed to check for existence of file at path[%s]: %s", filePath, err)
	}
	if fi.IsDir() {
		return false, "", fmt.Errorf("failed to get a file at path[%s]: got a directory", filePath)
	}
	return true, filePath, nil
}
