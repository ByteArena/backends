package main

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

type PerceptionSpecs struct {
	Weight int
	// statique
	// TBD
}

type PerceptionExternal struct {
	Vision int       // TBD
	Sound  []Vector2 // tableau de vecteurs (volume et direction) dans un espace quantisé
	Touch  int       // TBD; collisions ?
	Time   int       // en ms depuis le début de la partie
	Radar  int       // TBD; perception des obstacles ? position, vélocité, nature; position: segment 1d obstruant l'horizon 1D pour un monde 2D (à la Super hexagon) ?
	Xray   int       // TBD; vision à travers les obstacles
}

type PerceptionInternal struct {
	Energy           float64 // niveau en millièmes; reconstitution automatique ?
	Proprioception   float64 // surface occupée par le corps en rayon par rapport au centre géométrique
	Temperature      float64 // en degrés
	Balance          Vector2 // vecteur de longeur 1 pointant depuis le centre de gravité vers la négative du vecteur gravité
	Acceleration     Vector2 // vecteur de force (direction, magnitude)
	Gravity          Vector2 // vecteur de force (direction, magnitude)
	Damage           float64 // fiabilité générale en millièmes, fiabilité par système en millièmes
	Magnetoreception float64 // azimuth en degrés par rapport au "Nord" de l'arène
}

type PerceptionObjective struct {
	// TBD
	// mission ?
	// sens de la course ?
	// port du flag ou non ?
	// position du flag ?
}

type Perception struct {
	Specs     PerceptionSpecs
	External  PerceptionExternal
	Internal  PerceptionInternal
	Objective PerceptionObjective
}

func (agent *Agent) GetPerception() Perception {
	return Perception{}
}

func (agent *Agent) GetState() *AgentState {
	return agent.swarm.state.agents[agent.id]
}
