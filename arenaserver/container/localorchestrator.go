package container

import (
	"bufio"
	"context"
	"errors"

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
	events       chan interface{}
}

func (orch *LocalContainerOrchestrator) startContainerLocalOrch(ctner *arenaservertypes.AgentContainer, addTearDownCall func(commonTypes.TearDownCallback)) error {

	err := orch.cli.ContainerStart(
		orch.ctx,
		ctner.Containerid,
		types.ContainerStartOptions{},
	)

	if err != nil {
		return err
	}

	err = orch.localLogsToStdOut(ctner)

	if err != nil {
		return errors.New("Failed to follow docker container logs for " + ctner.Containerid)
	}

	containerInfo, err := orch.cli.ContainerInspect(
		orch.ctx,
		ctner.Containerid,
	)
	if err != nil {
		return errors.New("Could not inspect container " + ctner.Containerid)
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
		events:       make(chan interface{}),
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

		reader, err := orch.cli.ContainerLogs(orch.ctx, container.Containerid, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Details:    false,
			Timestamps: false,
		})

		utils.Check(err, "Could not read container logs for "+container.AgentId.String()+"; container="+container.Containerid)

		defer reader.Close()
		r := bufio.NewReader(reader)

		for {
			buf, _ := utils.ReadFullLine(r)
			if buf != "" {
				orch.events <- EventAgentLog{buf}
			}
		}

	}(orch, container)

	return nil
}

func (orch *LocalContainerOrchestrator) StartAgentContainer(ctner *arenaservertypes.AgentContainer, addTearDownCall func(t.TearDownCallback)) error {
	orch.events <- EventDebug{"Spawning agent " + ctner.AgentId.String()}

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
	orch.cli.ContainerKill(orch.ctx, container.Containerid, "KILL")
	//}
}

func (orch *LocalContainerOrchestrator) RemoveAgentContainer(ctner *arenaservertypes.AgentContainer) error {

	// We don't want to remove images in local mode
	return nil
}

func (orch *LocalContainerOrchestrator) Wait(ctner arenaservertypes.AgentContainer) (<-chan container.ContainerWaitOKBody, <-chan error) {
	waitChan, errorChan := orch.cli.ContainerWait(
		orch.ctx,
		ctner.Containerid,
		container.WaitConditionRemoved,
	)

	return waitChan, errorChan
}

func (orch *LocalContainerOrchestrator) SetAgentLogger(container *arenaservertypes.AgentContainer) error {
	// TODO(sven): implement log to stdout here
	return nil
}

func (orch *LocalContainerOrchestrator) TearDownAll() error {
	for _, container := range orch.containers {
		orch.TearDown(container)
	}

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

func (orch *LocalContainerOrchestrator) Events() chan interface{} {
	return orch.events
}
