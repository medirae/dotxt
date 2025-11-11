package config

import (
	"bytes"
	"dotxt/logging"
	"dotxt/terrors"
	"dotxt/utils"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	EnvPrefix = "DOTXT"
	EnvCFG    = "DOTXT_CONFIG"
)

var (
	DefaultPath = "~/.config/dotxt"
	configPath  string
	Color       bool
	Quiet       bool
)

func ConfigPath() string {
	return configPath
}
func setConfigPath(path string) error {
	path, err := utils.NormalizePath(path)
	if err != nil {
		return err
	}
	configPath = path
	return nil
}

func ConfigFile() string {
	return filepath.Join(ConfigPath(), "dotxt.toml")
}

func SelectConfigFile(arg string) error {
	var path string
	env := os.Getenv(EnvCFG)
	if arg != "" {
		path = arg
	} else if env != "" {
		path = env
	} else {
		path = DefaultPath
	}

	setConfigPath(path)
	return nil
}

func InitViper(arg string) error {
	err := SelectConfigFile(arg)
	if err != nil {
		return err
	}
	path := ConfigPath()
	viper.SetConfigType("toml")
	viper.SetConfigName("dotxt")
	viper.AddConfigPath(path)
	viper.SetEnvPrefix(EnvPrefix)
	viper.AutomaticEnv()

	err = viper.ReadConfig(bytes.NewReader([]byte(DefaultConfig)))
	if err != nil {
		return fmt.Errorf("%w: default configurations: %w", terrors.ErrParse, err)
	}
	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	errs := validateConfig()
	if len(errs) > 0 {
		for _, err := range errs {
			logging.Logger.Error(err)
		}
		os.Exit(2)
	}
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	err = viper.SafeWriteConfigAs(filepath.Join(path, "dotxt.toml"))
	if _, ok := err.(viper.ConfigFileAlreadyExistsError); ok {
		return nil
	}
	return err
}
