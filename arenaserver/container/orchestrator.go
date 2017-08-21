package container

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"

	"github.com/bytearena/bytearena/common/utils"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	uuid "github.com/satori/go.uuid"
)

type TearDownCallback func()

type ContainerOrchestrator struct {
	ctx               context.Context
	cli               *client.Client
	registryAuth      string
	containers        []AgentContainer
	GetHost           func(orch *ContainerOrchestrator) (string, error)
	StartContainer    func(orch *ContainerOrchestrator, ctner AgentContainer) error
	TearDownCallbacks []TearDownCallback
}

func (orch *ContainerOrchestrator) StartAgentContainer(ctner AgentContainer) error {
	utils.Debug("orch", "Spawning agent "+ctner.AgentId.String())

	return orch.StartContainer(orch, ctner)
}

func (orch *ContainerOrchestrator) RemoveAgentContainer(ctner AgentContainer) error {
	utils.Debug("orch", "Remove agent image "+ctner.ImageName)

	out, errImageRemove := orch.cli.ImageRemove(
		orch.ctx,
		ctner.ImageName,
		types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		},
	)

	utils.Debug("orch", "Removed "+strconv.Itoa(len(out))+" layers")

	return errImageRemove
}

func (orch *ContainerOrchestrator) Wait(ctner AgentContainer) {
	orch.cli.ContainerWait(
		orch.ctx,
		ctner.containerid.String(),
		container.WaitConditionRemoved,
	)
}

func (orch *ContainerOrchestrator) AddTearDownCall(fn TearDownCallback) {
	orch.TearDownCallbacks = append(orch.TearDownCallbacks, fn)
}

func (orch *ContainerOrchestrator) TearDown(container AgentContainer) {
	for _, cb := range orch.TearDownCallbacks {
		cb()
	}

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

	err := orch.RemoveAgentContainer(container)

	if err != nil {
		utils.Debug("orch", "Cannot remove agent container: "+err.Error())
	}
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

	containerUnixUser := os.Getenv("CONTAINER_UNIX_USER")

	if containerUnixUser == "" {
		containerUnixUser = "root"
	}

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
		reader, err := orch.cli.ImagePull(
			orch.ctx,
			dockerimage,
			types.ImagePullOptions{
				RegistryAuth: orch.registryAuth,
			},
		)

		if err != nil {
			return AgentContainer{}, errors.New("Failed to pull " + dockerimage + " from registry; " + err.Error())
		}

		defer reader.Close()

		io.Copy(os.Stdout, reader)
	}

	containerconfig := container.Config{
		Image: normalizedDockerimage,
		User:  containerUnixUser,
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
		NetworkMode:    "bridge",
		// Resources: container.Resources{
		// 	Memory: 1024 * 1024 * 32, // 32M
		// 	//CPUQuota: 5 * (1000),       // 5% en cent-millièmes
		// 	//CPUShares: 1,
		// 	CPUPercent: 5,
		// },
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

	agentcontainer := MakeAgentContainer(agentid, ContainerId(resp.ID), normalizedDockerimage)
	orch.containers = append(orch.containers, agentcontainer)

	return agentcontainer, nil
}
