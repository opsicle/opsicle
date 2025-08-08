package controller

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	CurrentSessionFilename = "current"
	CurrentSessionPath     = "/.opsicle/session"
)

func GetSessionTokenPath() (string, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user's home directory: %s", err)
	}
	sessionPath := filepath.Join(userHomeDir, CurrentSessionPath)
	fileInfo, err := os.Lstat(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(sessionPath, os.ModePerm); err != nil {
				return "", fmt.Errorf("failed to provision configuration directory at path[%s]: %s", sessionPath, err)
			}
			fileInfo, _ = os.Lstat(sessionPath)
		} else {
			return "", fmt.Errorf("path[%s] for session information does not exist: %s", sessionPath, err)
		}
	}
	if !fileInfo.IsDir() {
		return "", fmt.Errorf("path[%s] exists but is not a directory, it should be", sessionPath)
	}
	sessionFilePath := filepath.Join(sessionPath, CurrentSessionFilename)
	return sessionFilePath, nil
}

func GetSessionToken() (sessionToken string, sessionFilePath string, err error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to determine user's home directory: %s", err)
	}
	sessionPath := filepath.Join(userHomeDir, CurrentSessionPath)
	fileInfo, err := os.Lstat(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(sessionPath, os.ModePerm); err != nil {
				return "", "", fmt.Errorf("failed to provision configuration directory at path[%s]: %s", sessionPath, err)
			}
			fileInfo, _ = os.Lstat(sessionPath)
		} else {
			return "", "", fmt.Errorf("path[%s] for session information does not exist: %s", sessionPath, err)
		}
	}
	if !fileInfo.IsDir() {
		return "", "", fmt.Errorf("path[%s] exists but is not a directory, it should be", sessionPath)
	}
	sessionFilePath = filepath.Join(sessionPath, CurrentSessionFilename)
	fileInfo, err = os.Lstat(sessionFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("there is no current session")
		}
		return "", "", fmt.Errorf("failed to check current session file at path[%s]: %s", sessionFilePath, err)
	} else if fileInfo.IsDir() {
		return "", "", fmt.Errorf("path[%s] exists but is a directory, it should be a file", sessionFilePath)
	}
	sessionTokenData, err := os.ReadFile(sessionFilePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read file at path[%s]: %s", sessionFilePath, err)
	}
	sessionToken = string(sessionTokenData)
	return sessionToken, sessionFilePath, nil
}

func DeleteSessionToken() (err error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine user's home directory: %s", err)
	}
	sessionFilePath := filepath.Join(userHomeDir, CurrentSessionPath, CurrentSessionFilename)
	fileInfo, err := os.Lstat(sessionFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to check current session file at path[%s]: %s", sessionFilePath, err)
	} else if fileInfo.IsDir() {
		return fmt.Errorf("path[%s] exists but is a directory, it should be a file", sessionFilePath)
	}
	if err := os.Remove(sessionFilePath); err != nil {
		return fmt.Errorf("failed to remove session file at path[%s]: %s", sessionFilePath, err)
	}

	return nil
}
