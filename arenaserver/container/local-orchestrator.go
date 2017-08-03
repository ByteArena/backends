package container

import (
	"context"

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

func startContainerLocalOrch(orch *ContainerOrchestrator, ctner AgentContainer) error {

	return orch.cli.ContainerStart(
		orch.ctx,
		ctner.containerid.String(),
		types.ContainerStartOptions{},
	)
}

func MakeLocalContainerOrchestrator() ContainerOrchestrator {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.Check(err, "Failed to initialize docker client environment")

	registryAuth := ""

	return ContainerOrchestrator{
		ctx:            ctx,
		cli:            cli,
		registryAuth:   registryAuth,
		GetHost:        getHostLocalOrch,
		StartContainer: startContainerLocalOrch,
	}
}
