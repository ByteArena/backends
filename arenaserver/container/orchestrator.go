package container

import (
	"bufio"
	"context"
	"errors"
	"io/ioutil"
	"log"
	"strconv"

	"github.com/bytearena/bytearena/common/utils"
	"github.com/docker/distribution/reference"
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

type DockerRef struct {
	Registry string
	Path     string
	Tag      string
}

func normalizeDockerRef(dockerimage string) (string, error) {

	p, _ := reference.Parse(dockerimage)
	named, ok := p.(reference.Named)
	if !ok {
		return "", errors.New("Invalid docker image name")
	}

	parsedRefWithTag := reference.TagNameOnly(named)
	return parsedRefWithTag.String(), nil
}

func (orch *ContainerOrchestrator) CreateAgentContainer(agentid uuid.UUID, host string, port int, dockerimage string) (AgentContainer, error) {

	normalizedDockerimage, err := normalizeDockerRef(dockerimage)

	if err != nil {
		return AgentContainer{}, err
	}

	localimages, _ := orch.cli.ImageList(orch.ctx, types.ImageListOptions{})
	foundlocal := false
	for _, localimage := range localimages {
		for _, alias := range localimage.RepoTags {
			if normalizedAlias, err := normalizeDockerRef(alias); err == nil {
				if normalizedAlias == normalizedDockerimage {
					foundlocal = true
					break
				}
			}
		}

		if foundlocal {
			break
		}
	}

	if !foundlocal {
		rc, err := orch.cli.ImagePull(
			orch.ctx,
			dockerimage,
			types.ImagePullOptions{
				RegistryAuth: orch.registryAuth,
			},
		)

		if err != nil {
			return AgentContainer{}, errors.New("Failed to pull " + dockerimage + " from registry; " + err.Error())
		}

		defer rc.Close()
		ioutil.ReadAll(rc)
	}

	containerconfig := container.Config{
		Image: normalizedDockerimage,
		User:  "root",
		Env: []string{
			"PORT=" + strconv.Itoa(port),
			"HOST=" + host,
			"AGENTID=" + agentid.String(),
		},
		AttachStdout: false,
		AttachStderr: false,
	}
	
	log.Println("container config", containerconfig)

	hostconfig := container.HostConfig{
		CapDrop:        []string{"ALL"},
		Privileged:     false,
		AutoRemove:     true,
		ReadonlyRootfs: true,
		NetworkMode:    "host",
		//NetworkMode: "bridge",
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
		return AgentContainer{}, errors.New("Failed to create docker container for agent " + agentid.String() + "; " + err.Error())
	}

	agentcontainer := MakeAgentContainer(agentid, ContainerId(resp.ID))
	orch.containers = append(orch.containers, agentcontainer)

	return agentcontainer, nil
}

func (orch *ContainerOrchestrator) GetHost() (string, error) {
	res, err := orch.cli.NetworkInspect(orch.ctx, "bridge", types.NetworkInspectOptions{})
	if err != nil {
		return "", err
	}

	return res.IPAM.Config[0].Gateway, nil
}
