package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var Global global

type global struct {
	ControllerUrl string  `json:"controllerUrl" yaml:"controllerUrl"`
	SourcePath    *string `json:"sourcePath" yaml:"sourcePath"`
}

func (g *global) IsGlobalConfigExists() bool {
	return g.SourcePath != nil
}

func LoadGlobal(from string) error {

	globalConfigPath := from
	if !path.IsAbs(globalConfigPath) {
		if strings.Index(globalConfigPath, "~") == 0 {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to derive user home directory: %s", err)
			}
			globalConfigPath = filepath.Join(homeDir, globalConfigPath[1:])
		} else {
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to derive working directory: %s", err)
			}
			globalConfigPath = filepath.Join(workingDir, globalConfigPath)
		}
		logrus.Debugf("derived path[%s] from path[%s]", globalConfigPath, from)
	}
	logrus.Infof("loading global configuration from path[%s]...", globalConfigPath)

	isGlobalConfigLoaded := true
	fi, err := os.Stat(globalConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		logrus.Warnf("config file not found at path[%s], defaults will be used", globalConfigPath)
		isGlobalConfigLoaded = false
	} else if fi.IsDir() {
		logrus.Warnf("config file path[%s] led to a directory, defaults will be used", globalConfigPath)
		isGlobalConfigLoaded = false
	}
	viper.SetConfigFile(globalConfigPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read configuration file: %s", err)
	}
	if err := viper.Unmarshal(&Global); err != nil {
		return fmt.Errorf("failed to parse configuration file: %s", err)
	}
	if isGlobalConfigLoaded {
		Global.SourcePath = &globalConfigPath
	}

	return nil
}
