package container

import (
	"bufio"
	"context"
	"io/ioutil"
	"log"
	"strconv"

	"github.com/bytearena/bytearena/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

type ContainerOrchestrator struct {
	ctx          context.Context
	cli          *client.Client
	registryAuth string
	containers   []AgentContainer
}

func MakeContainerOrchestrator() ContainerOrchestrator {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.Check(err, "Failed to initialize docker client environment")

	registryAuth := registryLogin(ctx, cli)

	return ContainerOrchestrator{
		ctx:          ctx,
		cli:          cli,
		registryAuth: registryAuth,
	}
}

func (orch *ContainerOrchestrator) StartAgentContainer(ctner AgentContainer) error {

	log.Print(chalk.Yellow)
	log.Print("Spawning agent "+ctner.AgentId.String()+" in its own container", chalk.Reset)

	return orch.cli.ContainerStart(
		orch.ctx,
		ctner.containerid.String(),
		types.ContainerStartOptions{},
	)

}

func (orch *ContainerOrchestrator) Wait(ctner AgentContainer) error {
	orch.cli.ContainerWait(
		orch.ctx,
		ctner.containerid.String(),
		container.WaitConditionRemoved,
	)
	return nil
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

	}(orch, container)

	return nil
}

func (orch *ContainerOrchestrator) TearDown(container AgentContainer) {
	log.Println("TearDown !", container)

	// TODO: understand why this is sloooooooow since feat-build-git
	/*
		timeout := time.Second * 5
		err := orch.cli.ContainerStop(
			orch.ctx,
			container.containerid.String(),
			&timeout,
		)*/

	//if err != nil {
	orch.cli.ContainerKill(orch.ctx, container.containerid.String(), "KILL")
	//}
}

func (orch *ContainerOrchestrator) TearDownAll() {
	for _, container := range orch.containers {
		orch.TearDown(container)
	}
}

func (orch *ContainerOrchestrator) CreateAgentContainer(agentid uuid.UUID, host string, port int, dockerimage string) (AgentContainer, error) {

	rc, err := orch.cli.ImagePull(
		orch.ctx,
		dockerimage,
		types.ImagePullOptions{
			RegistryAuth: orch.registryAuth,
		},
	)

	utils.Assert(rc != nil, "Could not find docker image '"+dockerimage+"'")
	defer rc.Close()
	ioutil.ReadAll(rc)

	utils.Check(err, "Failed to pull "+dockerimage+" from registry")

	containerconfig := container.Config{
		Image: dockerimage,
		User:  "root",
		Env: []string{
			"PORT=" + strconv.Itoa(port),
			"HOST=" + host,
			"AGENTID=" + agentid.String(),
		},
		AttachStdout: false,
		AttachStderr: false,
	}

	hostconfig := container.HostConfig{
		CapDrop:        []string{"ALL"},
		Privileged:     false,
		AutoRemove:     true,
		ReadonlyRootfs: true,
		//NetworkMode:    "host",
		NetworkMode: "bridge",
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
	utils.Check(err, "Failed to create docker container for agent "+agentid.String())

	agentcontainer := MakeAgentContainer(agentid, ContainerId(resp.ID))
	orch.containers = append(orch.containers, agentcontainer)

	return agentcontainer, nil
}

func (orch *ContainerOrchestrator) GetHost() (string, error) {
	res, err := orch.cli.NetworkInspect(orch.ctx, "bridge", true)
	if err != nil {
		return "", err
	}

	return res.IPAM.Config[0].Gateway, nil
}
