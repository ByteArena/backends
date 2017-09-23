package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	uuid "github.com/satori/go.uuid"

	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	t "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

var logDir = utils.GetenvOrDefault("AGENT_LOGS_PATH", "./data/agent-logs")

type RemoteContainerOrchestrator struct {
	ctx          context.Context
	cli          *client.Client
	registryAuth string
	arenaAddr    string
	containers   []*arenaservertypes.AgentContainer
}

func (orch *RemoteContainerOrchestrator) startContainerRemoteOrch(ctner *arenaservertypes.AgentContainer, addTearDownCall func(t.TearDownCallback)) error {

	err := orch.cli.ContainerStart(
		orch.ctx,
		ctner.Containerid.String(),
		types.ContainerStartOptions{},
	)

	if err != nil {
		return err
	}

	err = orch.SetAgentLogger(ctner)

	if err != nil {
		return errors.New("Failed to follow docker container logs for " + ctner.Containerid.String())
	}

	addTearDownCall(func() error {
		utils.Debug("orch", "Closed agent container logger")

		ctner.LogReader.Close()

		ctner.LogWriter.Sync()
		ctner.LogWriter.Close()

		return nil
	})

	containerInfo, err := orch.cli.ContainerInspect(
		orch.ctx,
		ctner.Containerid.String(),
	)
	if err != nil {
		return errors.New("Could not inspect container " + ctner.Containerid.String())
	}

	ctner.SetIPAddress(containerInfo.NetworkSettings.IPAddress)

	return nil
}

func MakeRemoteContainerOrchestrator(arenaAddr string, registryAddr string) arenaservertypes.ContainerOrchestrator {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.Check(err, "Failed to initialize docker client environment")

	registryAuth := registryLogin(registryAddr, ctx, cli)

	return &RemoteContainerOrchestrator{
		ctx:          ctx,
		cli:          cli,
		registryAuth: registryAuth,
		arenaAddr:    arenaAddr,
	}
}

func (orch *RemoteContainerOrchestrator) GetHost() (string, error) {
	return orch.arenaAddr, nil
}

func (orch *RemoteContainerOrchestrator) StartAgentContainer(ctner *arenaservertypes.AgentContainer, addTearDownCall func(t.TearDownCallback)) error {
	utils.Debug("orch", "Spawning agent "+ctner.AgentId.String())

	return orch.startContainerRemoteOrch(ctner, addTearDownCall)
}

func (orch *RemoteContainerOrchestrator) SetAgentLogger(container *arenaservertypes.AgentContainer) error {

	go func(orch *RemoteContainerOrchestrator, container *arenaservertypes.AgentContainer) {
		reader, err := orch.cli.ContainerLogs(orch.ctx, container.Containerid.String(), types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Details:    false,
			Timestamps: false,
		})

		utils.Check(err, "Could not read container logs for "+container.AgentId.String()+"; container="+container.Containerid.String())

		// Create log file
		filename := logDir + "/" + container.AgentId.String() + ".log"
		utils.Debug("agent-logs", "created file "+filename)

		handle, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0777)

		container.SetLogger(reader, handle)

		_, err = io.Copy(handle, reader)
	}(orch, container)

	return nil
}

func (orch *RemoteContainerOrchestrator) CreateAgentContainer(agentid uuid.UUID, host string, port int, dockerimage string) (*arenaservertypes.AgentContainer, error) {
	return commonCreateAgentContainer(orch, agentid, host, port, dockerimage)
}

func (orch *RemoteContainerOrchestrator) TearDown(container *arenaservertypes.AgentContainer) {
	orch.cli.ContainerKill(orch.ctx, container.Containerid.String(), "KILL")

	err := orch.RemoveAgentContainer(container)
	if err != nil {
		utils.Debug("orch", "Cannot remove agent container: "+err.Error())
	}
}

func (orch *RemoteContainerOrchestrator) RemoveAgentContainer(ctner *arenaservertypes.AgentContainer) error {
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

func (orch *RemoteContainerOrchestrator) Wait(ctner arenaservertypes.AgentContainer) (<-chan container.ContainerWaitOKBody, <-chan error) {
	waitChan, errorChan := orch.cli.ContainerWait(
		orch.ctx,
		ctner.Containerid.String(),
		container.WaitConditionRemoved,
	)

	return waitChan, errorChan
}

func (orch *RemoteContainerOrchestrator) TearDownAll() error {
	for _, container := range orch.containers {
		orch.TearDown(container)
	}

	return nil
}

func (orch *RemoteContainerOrchestrator) GetCli() *client.Client {
	return orch.cli
}

func (orch *RemoteContainerOrchestrator) GetContext() context.Context {
	return orch.ctx
}

func (orch *RemoteContainerOrchestrator) GetRegistryAuth() string {
	return orch.registryAuth
}

func (orch *RemoteContainerOrchestrator) AddContainer(ctner *arenaservertypes.AgentContainer) {
	orch.containers = append(orch.containers, ctner)
}
