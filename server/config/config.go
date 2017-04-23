package config

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"path"

	"github.com/kardianos/osext"
)

type AgentGameConfig struct {
	Image string
}

type GameConfig struct {
	Port     int
	Tps      int
	Agentdir string
	Agents   []AgentGameConfig
}

type fileServerConfig struct {
	Server struct {
		Port     int
		Tps      int
		Agentdir string
	}
	Agents []struct {
		Scale int
		Git   string
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
	//assertString(config.Server.Host, "Host must be provided in the configuration")
	assertString(config.Server.Agentdir, "Agentdir must be provided in the configuration")

	gameconfig := GameConfig{
		Tps:      config.Server.Tps,
		Port:     config.Server.Port,
		Agentdir: getAbsoluteDir(config.Server.Agentdir),
	}

	for _, agentconfig := range config.Agents {
		config := createAgentGameConfig(agentconfig.Git)

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

func createAgentGameConfig(git string) AgentGameConfig {
	imageName := HashGitRepoName(git)

	return AgentGameConfig{
		Image: imageName,
	}
}

func getAbsoluteDir(relative string) string {

	exfolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	return path.Join(exfolder, relative)
}

func HashGitRepoName(git string) string {
	return getMD5Hash(git)
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
