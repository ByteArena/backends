package container

import (
	"bufio"
	"context"
	"errors"
	"log"

	t "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/ttacon/chalk"
)

func getHostLocalOrch(orch *ContainerOrchestrator) (string, error) {

	res, err := orch.cli.NetworkInspect(orch.ctx, "bridge", types.NetworkInspectOptions{})
	if err != nil {
		return "", err
	}

	return res.IPAM.Config[0].Gateway, nil
}

func startContainerLocalOrch(orch *ContainerOrchestrator, ctner *AgentContainer, addTearDownCall func(t.TearDownCallback)) error {

	err := orch.cli.ContainerStart(
		orch.ctx,
		ctner.containerid.String(),
		types.ContainerStartOptions{},
	)

	if err != nil {
		return err
	}

	err = localLogsToStdOut(orch, ctner)

	if err != nil {
		return errors.New("Failed to follow docker container logs for " + ctner.containerid.String())
	}

	containerInfo, err := orch.cli.ContainerInspect(
		orch.ctx,
		ctner.containerid.String(),
	)
	if err != nil {
		return errors.New("Could not inspect container " + ctner.containerid.String())
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

func localLogsToStdOut(orch *ContainerOrchestrator, container *AgentContainer) error {
	go func(orch *ContainerOrchestrator, container *AgentContainer) {
		reader, err := orch.cli.ContainerLogs(orch.ctx, container.containerid.String(), types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Details:    false,
			Timestamps: false,
		})

		utils.Check(err, "Could not read container logs for "+container.AgentId.String()+"; container="+container.containerid.String())

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
