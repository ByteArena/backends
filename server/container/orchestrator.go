package container

import (
	"bufio"
	"context"
	"log"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/netgusto/bytearena/server/config"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

type ContainerOrchestrator struct {
	ctx        context.Context
	cli        *client.Client
	containers []AgentContainer
}

func MakeContainerOrchestrator() ContainerOrchestrator {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Panicln(err)
	}

	return ContainerOrchestrator{
		ctx: ctx,
		cli: cli,
	}
}

func (orch *ContainerOrchestrator) StartAgentContainer(container AgentContainer) error {

	log.Print(chalk.Yellow)
	log.Print("Spawning agent "+container.AgentId.String()+" in its own container", chalk.Reset)

	return orch.cli.ContainerStart(
		orch.ctx,
		container.containerid.String(),
		types.ContainerStartOptions{},
	)

}

func (orch *ContainerOrchestrator) Wait(container AgentContainer) error {
	_, err := orch.cli.ContainerWait(
		orch.ctx,
		container.containerid.String(),
	)
	return err
}

func (orch *ContainerOrchestrator) LogsToStdOut(container AgentContainer) error {
	go func(orch *ContainerOrchestrator, container AgentContainer) {
		reader, _ := orch.cli.ContainerLogs(orch.ctx, container.containerid.String(), types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Details:    false,
			Timestamps: false,
		})
		defer reader.Close()

		r := bufio.NewReader(reader)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			log.Println(chalk.Green, container.AgentId, chalk.Reset, text)
		}

		//p := make([]byte, 8)
		//reader.Read(p)
		//content, _ := ioutil.ReadAll(reader)
		//log.Println("CONTAINER LOG", string(content))
	}(orch, container)

	return nil
}

func (orch *ContainerOrchestrator) TearDown(container AgentContainer) {

	timeout := time.Second * 5
	err := orch.cli.ContainerStop(
		orch.ctx,
		container.containerid.String(),
		&timeout,
	)

	if err != nil {
		orch.cli.ContainerKill(orch.ctx, container.containerid.String(), "KILL")
	}
}

func (orch *ContainerOrchestrator) TearDownAll() {
	for _, container := range orch.containers {
		orch.TearDown(container)
	}
}

func (orch *ContainerOrchestrator) CreateAgentContainer(agentid uuid.UUID, host string, port int, agentdir string, config config.AgentGameConfig) (AgentContainer, error) {

	containerconfig := container.Config{
		Image: config.Image,
		Cmd:   []string{"/bin/bash", "-c", config.Cmd},
		User:  "root",
		Env: []string{
			"SWARMPORT=" + strconv.Itoa(port),
			"SWARMHOST=" + host,
			"AGENTID=" + agentid.String(),
		},
		AttachStdout: false,
		AttachStderr: false,
	}

	hostconfig := container.HostConfig{
		CapDrop:        []string{"ALL"},
		Privileged:     false,
		Binds:          []string{config.Dir + ":/scripts"}, // SCRIPTPATH references file path on docker host, not on current container
		AutoRemove:     true,
		ReadonlyRootfs: true,
		NetworkMode:    "host",
		Resources: container.Resources{
			Memory: 1024 * 1024 * 32, // 32M
			//CPUQuota: 5 * (1000),       // 5% en cent-millièmes
			//CPUShares: 1,
			CPUPercent: 5,
		},
	}

	resp, err := orch.cli.ContainerCreate(
		orch.ctx,         // go context
		&containerconfig, // container config
		&hostconfig,      // host config
		nil,              // network config
		"agent-"+agentid.String(), // container name
	)
	if err != nil {
		log.Panicln(err)
	}

	agentcontainer := MakeAgentContainer(agentid, ContainerId(resp.ID))
	orch.containers = append(orch.containers, agentcontainer)

	return agentcontainer, nil
}
