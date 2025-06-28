package config

import (
	"errors"
	"fmt"
	"os"

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
	logrus.Infof("loading global configuration from path[%s]...", from)

	isGlobalConfigLoaded := true
	fi, err := os.Stat(from)
	if errors.Is(err, os.ErrNotExist) {
		logrus.Warnf("config file not found at path[%s], defaults will be used", from)
		isGlobalConfigLoaded = false
	} else if fi.IsDir() {
		logrus.Warnf("config file path[%s] led to a directory, defaults will be used", from)
		isGlobalConfigLoaded = false
	}
	viper.SetConfigFile(from)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read configuration file: %s", err)
	}
	if err := viper.Unmarshal(&Global); err != nil {
		return fmt.Errorf("failed to parse configuration file: %s", err)
	}
	if isGlobalConfigLoaded {
		Global.SourcePath = &from
	}

	return nil
}
