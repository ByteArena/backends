package server

import (
	"bytes"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

type Agent struct {
	id          uuid.UUID
	containerid string
	swarm       *Swarm
	tcp         *TCPClient
}

func NewAgent(swarm *Swarm) *Agent {

	// random uuid
	agentid := uuid.NewV4()

	containerconfig := container.Config{
		Image: "node",
		Cmd:   []string{"/bin/bash", "-c", "node --harmony /scripts/client.js"},
		User:  "node",
		Env: []string{
			"SWARMPORT=" + strconv.Itoa(swarm.port),
			"SWARMHOST=" + swarm.host,
			"AGENTID=" + agentid.String(),
		},
	}

	hostconfig := container.HostConfig{
		CapDrop:    []string{"ALL"},
		Privileged: false,
		Binds:      []string{swarm.agentdir + ":/scripts"}, // SCRIPTPATH references file path on docker host, not on current container
		Resources: container.Resources{
			Memory:   1024 * 1024 * 32,   // 32M
			CPUQuota: 5 * (100000 / 100), // 5%
		},
	}

	resp, err := swarm.cli.ContainerCreate(
		swarm.ctx,        // go context
		&containerconfig, // container config
		&hostconfig,      // host config
		nil,              // network config
		"agent-"+agentid.String(), // container name
	)
	if err != nil {
		log.Panicln(err)
	}

	return &Agent{
		id:          agentid,
		swarm:       swarm,
		containerid: resp.ID,
	}

}

func (agent *Agent) Start() error {

	log.Print(chalk.Yellow)
	log.Print("Spawning agent "+agent.id.String()+" in its own container", chalk.Reset)
	log.Println("")

	return agent.swarm.cli.ContainerStart(
		agent.swarm.ctx,
		agent.containerid,
		types.ContainerStartOptions{},
	)
}

func (agent *Agent) Wait() error {
	_, err := agent.swarm.cli.ContainerWait(
		agent.swarm.ctx,
		agent.containerid,
	)
	return err
}

func (agent *Agent) Stop() {
	timeout := time.Second * 10
	err := agent.swarm.cli.ContainerStop(
		agent.swarm.ctx,
		agent.containerid,
		&timeout,
	)
	if err != nil {
		agent.Kill()
	}
}

func (agent *Agent) Kill() {
	agent.swarm.cli.ContainerKill(agent.swarm.ctx, agent.containerid, "KILL")
}

func (agent *Agent) Logs() (io.ReadCloser, error) {
	return agent.swarm.cli.ContainerLogs(
		agent.swarm.ctx,
		agent.containerid,
		types.ContainerLogsOptions{ShowStdout: true},
	)
}

func (agent *Agent) LogsToString() (string, error) {
	out, err := agent.Logs()
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(out)

	return buf.String(), nil
}

func (agent *Agent) Remove() error {
	return agent.swarm.cli.ContainerRemove(
		agent.swarm.ctx,
		agent.containerid,
		types.ContainerRemoveOptions{},
	)
}

func (agent *Agent) Teardown() {
	agent.Stop()
	err := agent.Remove()
	if err != nil {
		log.Panicln(err)
	}
}

func (agent *Agent) GetPerception() Perception {
	p := Perception{}
	agentstate := agent.GetState()
	//	p.Internal.Acceleration = agentstate.Acceleration.clone()
	p.Internal.Velocity = agentstate.Velocity.Clone()
	p.Internal.Proprioception = agentstate.Radius

	// On rend la position de l'attractor relative à l'agent
	p.Objective.Attractor = agent.swarm.state.Pin.Clone().Sub(agentstate.Position)

	p.Specs.MaxSpeed = 8
	p.Specs.MaxSteeringForce = 4

	return p
}

func (agent *Agent) GetState() *AgentState {
	return agent.swarm.state.Agents[agent.id]
}
