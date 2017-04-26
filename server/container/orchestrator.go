package container

import (
	"bufio"
	"context"
	"log"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/netgusto/bytearena/server/config"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

type ContainerOrchestrator struct {
	ctx          context.Context
	cli          *client.Client
	registryAuth string
	containers   []AgentContainer
}

func MakeContainerOrchestrator() ContainerOrchestrator {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	utils.Check(err, "Failed to initialize docker client environment")

	registryAuth := registryLogin(ctx, cli)

	return ContainerOrchestrator{
		ctx:          ctx,
		cli:          cli,
		registryAuth: registryAuth,
	}
}

func (orch *ContainerOrchestrator) StartAgentContainer(container AgentContainer) error {

	log.Print(chalk.Yellow)
	log.Print("Spawning agent "+container.AgentId.String()+" in its own container", chalk.Reset)

	return orch.cli.ContainerStart(
		orch.ctx,
		container.containerid.String(),
		types.ContainerStartOptions{},
	)

}

func (orch *ContainerOrchestrator) Wait(container AgentContainer) error {
	_, err := orch.cli.ContainerWait(
		orch.ctx,
		container.containerid.String(),
	)
	return err
}

func (orch *ContainerOrchestrator) LogsToStdOut(container AgentContainer) error {
	go func(orch *ContainerOrchestrator, container AgentContainer) {
		reader, _ := orch.cli.ContainerLogs(orch.ctx, container.containerid.String(), types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Details:    false,
			Timestamps: false,
		})
		defer reader.Close()

		r := bufio.NewReader(reader)
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			log.Println(chalk.Green, container.AgentId, chalk.Reset, text)
		}

		//p := make([]byte, 8)
		//reader.Read(p)
		//content, _ := ioutil.ReadAll(reader)
		//log.Println("CONTAINER LOG", string(content))
	}(orch, container)

	return nil
}

func (orch *ContainerOrchestrator) TearDown(container AgentContainer) {

	timeout := time.Second * 5
	err := orch.cli.ContainerStop(
		orch.ctx,
		container.containerid.String(),
		&timeout,
	)

	if err != nil {
		orch.cli.ContainerKill(orch.ctx, container.containerid.String(), "KILL")
	}
}

func (orch *ContainerOrchestrator) TearDownAll() {
	for _, container := range orch.containers {
		orch.TearDown(container)
	}
}

func (orch *ContainerOrchestrator) CreateAgentContainer(agentid uuid.UUID, host string, port int, config config.AgentGameConfig) (AgentContainer, error) {

	//config.Image = "127.0.0.1:5000/bytearena_bar:latest"
	config.Image = "127.0.0.1:5000/10c09925f84312b02486f93a41f32e58" // Sven, ceci est mon ID local; change-le quand tu auras un panic !

	_, err := orch.cli.ImagePull(
		orch.ctx,
		config.Image,
		types.ImagePullOptions{
			RegistryAuth: orch.registryAuth,
		},
	)

	utils.Check(err, "Failed to pull "+config.Image+" from registry")

	containerconfig := container.Config{
		Image: config.Image,
		User:  "root",
		Env: []string{
			"SWARMPORT=" + strconv.Itoa(port),
			"SWARMHOST=" + host,
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
		//NetworkMode:    "host",
		Links:       nil,
		NetworkMode: "bridge",
		Resources: container.Resources{
			Memory: 1024 * 1024 * 32, // 32M
			//CPUQuota: 5 * (1000),       // 5% en cent-millièmes
			//CPUShares: 1,
			CPUPercent: 5,
		},
	}

	resp, err := orch.cli.ContainerCreate(
		orch.ctx,         // go context
		&containerconfig, // container config
		&hostconfig,      // host config
		nil,              // network config
		"agent-"+agentid.String(), // container name
	)
	utils.Check(err, "Failed to create docker container for agent "+agentid.String())

	agentcontainer := MakeAgentContainer(agentid, ContainerId(resp.ID))
	orch.containers = append(orch.containers, agentcontainer)

	return agentcontainer, nil
}

func (orch *ContainerOrchestrator) GetHost() (string, error) {
	res, err := orch.cli.NetworkInspect(orch.ctx, "bridge", true)
	if err != nil {
		return "", err
	}

	return res.IPAM.Config[0].Gateway, nil
}
