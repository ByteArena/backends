package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"

	"github.com/kardianos/osext"
)

type AgentGameConfig struct {
	Cmd   string
	Dir   string
	Image string
}

type GameConfig struct {
	Agents   []AgentGameConfig
	Port     int
	Agentdir string
	Host     string
	Tps      int
}

type fileServerConfig struct {
	Server struct {
		Port     int
		Tps      int
		Host     string
		Agentdir string
	}
	Agents []struct {
		Scale int
		Dir   string
	}
}

func LoadServerConfig(filename string) GameConfig {
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Panicln(err)
	}

	var config fileServerConfig

	if err := json.Unmarshal(data, &config); err != nil {
		log.Panicln(err)
	}

	assertInt(config.Server.Port, "Port number must be provided in the configuration")
	assertInt(config.Server.Tps, "TPS must be provided in the configuration")
	assertString(config.Server.Host, "Host must be provided in the configuration")
	assertString(config.Server.Agentdir, "Agentdir must be provided in the configuration")

	gameconfig := GameConfig{
		Tps:      config.Server.Tps,
		Port:     config.Server.Port,
		Host:     config.Server.Host,
		Agentdir: getAbsoluteDir(config.Server.Agentdir),
	}

	for _, agentconfig := range config.Agents {
		agentdir := path.Join(gameconfig.Agentdir, agentconfig.Dir)
		config := loadAgentConfig(agentdir + "/config.json")

		config.Dir = agentdir

		if agentconfig.Scale != 0 {

			i := 0
			for i < agentconfig.Scale {
				gameconfig.Agents = append(gameconfig.Agents, config)
				i++
			}
		} else {
			gameconfig.Agents = append(gameconfig.Agents, config)
		}
	}

	return gameconfig
}

func assertInt(value int, err string) {
	if value == 0 {
		log.Panic(err)
	}
}

func assertString(value string, err string) {
	if value == "" {
		log.Panic(err)
	}
}

func loadAgentConfig(filename string) AgentGameConfig {
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Panicln(err)
	}

	var config AgentGameConfig

	if err := json.Unmarshal(data, &config); err != nil {
		log.Panicln(err)
	}

	return config
}

func getAbsoluteDir(relative string) string {

	exfolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	return path.Join(exfolder, relative)
}
