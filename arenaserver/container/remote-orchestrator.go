package container

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/bytearena/bytearena/common/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func getHostRomoteOrch(orch *ContainerOrchestrator) (string, error) {

	// Inspecting the present container (arena-server) to get it's IP in the "agents" network
	res, err := orch.cli.ContainerInspect(
		orch.ctx,
		os.Getenv("HOSTNAME"), // $HOSTNAME is the container id
	)

	if err != nil {
		return "", err
	}

	ipInAgentsNetwork := ""

	for networkname, network := range res.NetworkSettings.Networks {
		if networkname == "agents" {
			ipInAgentsNetwork = network.IPAddress
			break
		}
	}

	if ipInAgentsNetwork == "" {
		return "", errors.New("Could not determine IP of arena-server container in the 'agents' network")
	}

	return ipInAgentsNetwork, nil
}

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

	return orch.cli.NetworkDisconnect(
		orch.ctx,
		defaultID,
		ctner.containerid.String(),
		true,
	)
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
		// GetHost:        getHostRomoteOrch,
		GetHost: func(orch *ContainerOrchestrator) (string, error) {
			return arenaAddr, nil
		},
		StartContainer: startContainerRemoteOrch,
	}
}
