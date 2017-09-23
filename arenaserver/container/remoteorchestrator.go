package container

import (
	"context"
	"errors"
	"io"
	"os"

	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	t "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var logDir = utils.GetenvOrDefault("AGENT_LOGS_PATH", "./data/agent-logs")

func startContainerRemoteOrch(orch *ContainerOrchestrator, ctner *arenaservertypes.AgentContainer, addTearDownCall func(t.TearDownCallback)) error {

	err := orch.cli.ContainerStart(
		orch.ctx,
		ctner.Containerid.String(),
		types.ContainerStartOptions{},
	)

	if err != nil {
		return err
	}

	err = setAgentLogger(orch, ctner)

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
		RemoveImages:   true,
	}
}

func setAgentLogger(orch *ContainerOrchestrator, container *arenaservertypes.AgentContainer) error {

	go func(orch *ContainerOrchestrator, container *arenaservertypes.AgentContainer) {
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
