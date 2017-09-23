package container

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	uuid "github.com/satori/go.uuid"

	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	commonTypes "github.com/bytearena/bytearena/common/types"
	t "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

type LocalContainerOrchestrator struct {
	ctx          context.Context
	cli          *client.Client
	registryAuth string
	host         string
	containers   []*arenaservertypes.AgentContainer
}

func (orch *LocalContainerOrchestrator) startContainerLocalOrch(ctner *arenaservertypes.AgentContainer, addTearDownCall func(commonTypes.TearDownCallback)) error {

	err := orch.cli.ContainerStart(
		orch.ctx,
		ctner.Containerid.String(),
		types.ContainerStartOptions{},
	)

	if err != nil {
		return err
	}

	err = orch.localLogsToStdOut(ctner)

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

func MakeLocalContainerOrchestrator(host string) arenaservertypes.ContainerOrchestrator {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.Check(err, "Failed to initialize docker client environment")

	registryAuth := ""

	return &LocalContainerOrchestrator{
		ctx:          ctx,
		cli:          cli,
		host:         host,
		registryAuth: registryAuth,
	}
}

func (orch *LocalContainerOrchestrator) GetHost() (string, error) {
	if orch.host == "" {
		res, err := orch.cli.NetworkInspect(orch.ctx, "bridge", types.NetworkInspectOptions{})
		if err != nil {
			return "", err
		}

		return res.IPAM.Config[0].Gateway, nil
	}

	return orch.host, nil
}

func (orch *LocalContainerOrchestrator) localLogsToStdOut(container *arenaservertypes.AgentContainer) error {
	go func(orch *LocalContainerOrchestrator, container *arenaservertypes.AgentContainer) {
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

func (orch *LocalContainerOrchestrator) StartAgentContainer(ctner *arenaservertypes.AgentContainer, addTearDownCall func(t.TearDownCallback)) error {
	utils.Debug("orch", "Spawning agent "+ctner.AgentId.String())

	return orch.startContainerLocalOrch(ctner, addTearDownCall)
}

func (orch *LocalContainerOrchestrator) CreateAgentContainer(agentid uuid.UUID, host string, port int, dockerimage string) (*arenaservertypes.AgentContainer, error) {
	return commonCreateAgentContainer(orch, agentid, host, port, dockerimage)
}

func (orch *LocalContainerOrchestrator) TearDown(container *arenaservertypes.AgentContainer) {
	// timeout := time.Second * 5
	// err := orch.cli.ContainerStop(
	// 	orch.ctx,
	// 	container.containerid.String(),
	// 	&timeout,
	// )

	// if err != nil {
	orch.cli.ContainerKill(orch.ctx, container.Containerid.String(), "KILL")
	//}

	err := orch.RemoveAgentContainer(container)
	if err != nil {
		utils.Debug("orch", "Cannot remove agent container: "+err.Error())
	}
}

func (orch *LocalContainerOrchestrator) RemoveAgentContainer(ctner *arenaservertypes.AgentContainer) error {
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

func (orch *LocalContainerOrchestrator) Wait(ctner arenaservertypes.AgentContainer) (<-chan container.ContainerWaitOKBody, <-chan error) {
	waitChan, errorChan := orch.cli.ContainerWait(
		orch.ctx,
		ctner.Containerid.String(),
		container.WaitConditionRemoved,
	)

	return waitChan, errorChan
}

func (orch *LocalContainerOrchestrator) SetAgentLogger(container *arenaservertypes.AgentContainer) error {
	// TODO(sven): implement log to stdout here
	return nil
}

func (orch *LocalContainerOrchestrator) TearDownAll() error {
	fmt.Println("Implement TearDownAll in LocalContainerOrchestrator")

	return nil
}

func (orch *LocalContainerOrchestrator) GetCli() *client.Client {
	return orch.cli
}

func (orch *LocalContainerOrchestrator) GetContext() context.Context {
	return orch.ctx
}

func (orch *LocalContainerOrchestrator) GetRegistryAuth() string {
	return orch.registryAuth
}

func (orch *LocalContainerOrchestrator) AddContainer(ctner *arenaservertypes.AgentContainer) {
	orch.containers = append(orch.containers, ctner)
}
