package agent

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/netgusto/bytearena/server/state"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"
)

type Agent struct {
	id          uuid.UUID
	containerid string
}

func NewAgent(ctx context.Context, cli *client.Client, port int, host string, agentdir string) *Agent {

	// random uuid
	agentid := uuid.NewV4()

	containerconfig := container.Config{
		Image: "node",
		Cmd:   []string{"/bin/bash", "-c", "node --harmony /scripts/client.js"},
		User:  "node",
		Env: []string{
			"SWARMPORT=" + strconv.Itoa(port),
			"SWARMHOST=" + host,
			"AGENTID=" + agentid.String(),
		},
	}

	hostconfig := container.HostConfig{
		CapDrop:    []string{"ALL"},
		Privileged: false,
		Binds:      []string{agentdir + ":/scripts"}, // SCRIPTPATH references file path on docker host, not on current container
		Resources: container.Resources{
			Memory:   1024 * 1024 * 32,   // 32M
			CPUQuota: 5 * (100000 / 100), // 5%
		},
	}

	resp, err := cli.ContainerCreate(
		ctx,              // go context
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
		containerid: resp.ID,
	}

}

func (agent *Agent) String() string {
	return "<Agent(" + agent.id.String() + ">"
}

func (agent *Agent) Start(ctx context.Context, cli *client.Client) error {

	log.Print(chalk.Yellow)
	log.Print("Spawning agent "+agent.id.String()+" in its own container", chalk.Reset)
	log.Println("")

	return cli.ContainerStart(
		ctx,
		agent.containerid,
		types.ContainerStartOptions{},
	)
}

func (agent *Agent) Wait(ctx context.Context, cli *client.Client) error {
	_, err := cli.ContainerWait(
		ctx,
		agent.containerid,
	)
	return err
}

func (agent *Agent) Stop(ctx context.Context, cli *client.Client) {
	timeout := time.Second * 10
	err := cli.ContainerStop(
		ctx,
		agent.containerid,
		&timeout,
	)
	if err != nil {
		agent.Kill(ctx, cli)
	}
}

func (agent *Agent) Kill(ctx context.Context, cli *client.Client) {
	cli.ContainerKill(ctx, agent.containerid, "KILL")
}

// func (agent *Agent) Logs() (io.ReadCloser, error) {
// 	return cli.ContainerLogs(
// 		ctx,
// 		agent.containerid,
// 		types.ContainerLogsOptions{ShowStdout: true},
// 	)
// }

// func (agent *Agent) LogsToString() (string, error) {
// 	out, err := agent.Logs()
// 	if err != nil {
// 		return "", err
// 	}

// 	buf := new(bytes.Buffer)
// 	buf.ReadFrom(out)

// 	return buf.String(), nil
// }

func (agent *Agent) Remove(
	ctx context.Context,
	cli *client.Client,
) error {

	return cli.ContainerRemove(
		ctx,
		agent.containerid,
		types.ContainerRemoveOptions{},
	)
}

func (agent *Agent) Teardown(ctx context.Context, cli *client.Client) {
	agent.Stop(ctx, cli)
	err := agent.Remove(ctx, cli)
	if err != nil {
		log.Panicln(err)
	}
}

func (agent *Agent) GetPerception() state.Perception {
	p := state.Perception{}
	return p
	// agentstate := agent.GetState()
	// //	p.Internal.Acceleration = agentstate.Acceleration.clone()
	// p.Internal.Velocity = agentstate.Velocity.Clone()
	// p.Internal.Proprioception = agentstate.Radius

	// // On rend la position de l'attractor relative à l'agent
	// p.Objective.Attractor = agent.swarm.state.Pin.Clone().Sub(agentstate.Position)

	// p.Specs.MaxSpeed = 8
	// p.Specs.MaxSteeringForce = 4

	// return p
}

func (agent *Agent) GetState() state.AgentState {
	// return agent.swarm.state.Agents[agent.id]
	return state.AgentState{}
}

func (agent *Agent) SetState(state state.AgentState) {
	// agent.swarm.state.Agents[agent.id] = state
}
