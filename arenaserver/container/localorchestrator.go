package container

import (
	"bufio"
	"context"
	"errors"

	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	commonTypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func getHostLocalOrch(orch *ContainerOrchestrator) (string, error) {

	res, err := orch.cli.NetworkInspect(orch.ctx, "bridge", types.NetworkInspectOptions{})
	if err != nil {
		return "", err
	}

	return res.IPAM.Config[0].Gateway, nil
}

func startContainerLocalOrch(orch *ContainerOrchestrator, ctner *arenaservertypes.AgentContainer, addTearDownCall func(commonTypes.TearDownCallback)) error {

	err := orch.cli.ContainerStart(
		orch.ctx,
		ctner.Containerid.String(),
		types.ContainerStartOptions{},
	)

	if err != nil {
		return err
	}

	err = localLogsToStdOut(orch, ctner)

	if err != nil {
		return errors.New("Failed to follow docker container logs for " + ctner.Containerid.String())
	}

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

func MakeLocalContainerOrchestrator(host string) ContainerOrchestrator {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.Check(err, "Failed to initialize docker client environment")

	registryAuth := ""

	return ContainerOrchestrator{
		ctx:          ctx,
		cli:          cli,
		registryAuth: registryAuth,
		GetHost: func(orch *ContainerOrchestrator) (string, error) {
			if host == "" {
				return getHostLocalOrch(orch)
			}

			return host, nil
		},
		StartContainer: startContainerLocalOrch,
		RemoveImages:   false,
	}
}

func localLogsToStdOut(orch *ContainerOrchestrator, container *arenaservertypes.AgentContainer) error {
	go func(orch *ContainerOrchestrator, container *arenaservertypes.AgentContainer) {
		reader, err := orch.cli.ContainerLogs(orch.ctx, container.Containerid.String(), types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Details:    false,
			Timestamps: false,
		})

		utils.Check(err, "Could not read container logs for "+container.AgentId.String()+"; container="+container.Containerid.String())

		defer reader.Close()

		r := bufio.NewReader(reader)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			utils.Debug(container.AgentId.String()+"/"+container.ImageName, text)
		}

	}(orch, container)

	return nil
}
