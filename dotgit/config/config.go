package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Config struct {
	DatabaseURI         string
	GitRepositoriesPath string
	MqHost              string

	DockerHost             string
	DockerBuildMemoryLimit string
	DockerBuildSwapLimit   string
	DockerBuildNetwork     string
	DockerBuildNoCache     string
	DockerBuildCpuPeriod   string
}

func (conf Config) GetDatabaseURI() string {
	return conf.DatabaseURI
}

func (conf Config) GetGitRepositoriesPath() string {
	return conf.GitRepositoriesPath
}

func (conf Config) GetMqHost() string {
	return conf.MqHost
}

func (conf Config) GetDockerHost() string {
	return conf.DockerHost
}

func loadConfig(configpath string) (*Config, error) {
	if _, err := os.Stat(configpath); os.IsNotExist(err) {
		return nil, errors.New("Missing config file: " + configpath)
	}

	buf, err := ioutil.ReadFile(configpath) // just pass the file name
	if err != nil {
		return nil, errors.New("Cannot read config file: " + configpath)
	}

	var config Config
	if err = json.Unmarshal(buf, &config); err != nil {
		return nil, errors.New("Invalid JSON in config file: " + configpath)
	}

	if strings.TrimSpace(config.GetDatabaseURI()) == "" {
		return nil, errors.New("DatabaseURI is missing in config file: " + configpath)
	}

	if strings.TrimSpace(config.GetGitRepositoriesPath()) == "" {
		return nil, errors.New("GitRepositoriesPath is missing in config file: " + configpath)
	}

	if strings.TrimSpace(config.GetMqHost()) == "" {
		return nil, errors.New("MqHost is missing in config file: " + configpath)
	}

	if strings.TrimSpace(config.GetDockerHost()) == "" {
		return nil, errors.New("DockerHost is missing in config file: " + configpath)
	}

	return &config, nil
}

var _cnf *Config

func GetConfig() *Config {

	if _cnf != nil {
		return _cnf
	}

	configpath, exists := os.LookupEnv("DOTGIT_CONFIG")
	if !exists {
		configpath = "/etc/dotgit.conf"
	}

	conf, err := loadConfig(configpath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	_cnf = conf

	return _cnf
}
