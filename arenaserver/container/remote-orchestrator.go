package container

import (
	"context"
	"errors"
	"io"
	"log"
	"os"

	"github.com/bytearena/bytearena/common/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var logDir = "/var/log/agents"

func startContainerRemoteOrch(orch *ContainerOrchestrator, ctner AgentContainer) error {

	err := orch.cli.ContainerStart(
		orch.ctx,
		ctner.containerid.String(),
		types.ContainerStartOptions{},
	)

	if err != nil {
		return err
	}

	networks, err := orch.cli.NetworkList(
		orch.ctx,
		types.NetworkListOptions{},
	)

	networkID := ""
	defaultID := ""

	for _, network := range networks {
		if network.Name == "agents" {
			networkID = network.ID
		} else if network.Name == "bridge" {
			defaultID = network.ID
		}
	}

	if networkID == "" {
		log.Panicln("CANNOT FIND AGENTS NETWORK !!")
	}

	if defaultID == "" {
		log.Panicln("CANNOT FIND DEFAULT NETWORK !!")
	}

	err = orch.cli.NetworkConnect(
		orch.ctx,
		networkID,
		ctner.containerid.String(),
		nil,
	)

	if err != nil {
		return err
	}

	err = orch.cli.NetworkDisconnect(
		orch.ctx,
		defaultID,
		ctner.containerid.String(),
		true,
	)

	if err != nil {
		return err
	}

	err = remoteLogsToSyslog(orch, ctner)

	if err != nil {
		return errors.New("Failed to follow docker container logs for " + ctner.containerid.String())
	}

	return nil
}

func MakeRemoteContainerOrchestrator(arenaAddr string, registryAddr string) ContainerOrchestrator {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.Check(err, "Failed to initialize docker client environment")

	registryAuth := registryLogin(registryAddr, ctx, cli)

	return ContainerOrchestrator{
		ctx:          ctx,
		cli:          cli,
		registryAuth: registryAuth,
		GetHost: func(orch *ContainerOrchestrator) (string, error) {
			return arenaAddr, nil
		},
		StartContainer: startContainerRemoteOrch,
	}
}

func remoteLogsToSyslog(orch *ContainerOrchestrator, container AgentContainer) error {
	go func(orch *ContainerOrchestrator, container AgentContainer) {
		reader, err := orch.cli.ContainerLogs(orch.ctx, container.containerid.String(), types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Details:    false,
			Timestamps: false,
		})

		utils.Check(err, "Could not read container logs for "+container.AgentId.String()+"; container="+container.containerid.String())

		defer reader.Close()

		// Create log file
		filename := logDir + "/" + container.AgentId.String() + ".log"
		utils.Debug("agent-logs", "created file "+filename)

		handle, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)

		_, err = io.Copy(handle, reader)

		handle.Sync()
		handle.Close()

	}(orch, container)

	return nil
}
