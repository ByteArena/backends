package agent

import (
	"github.com/netgusto/bytearena/server/state"
	uuid "github.com/satori/go.uuid"
)

type Agent struct {
	Id uuid.UUID
}

func MakeAgent() Agent {
	return Agent{
		Id: uuid.NewV4(), // random uuid
	}
}

func (agent *Agent) String() string {
	return "<Agent(" + agent.Id.String() + ")>"
}

func (agent *Agent) GetPerception(swarmstate *state.ServerState) state.Perception {
	p := state.Perception{}
	agentstate := agent.GetState(swarmstate)

	p.Internal.Velocity = agentstate.Velocity.Clone()
	p.Internal.Proprioception = agentstate.Radius

	// On rend la position de l'attractor relative à l'agent
	p.Objective.Attractor = swarmstate.Pin.Clone().Sub(agentstate.Position)

	p.Specs.MaxSpeed = agentstate.MaxSpeed
	p.Specs.MaxSteeringForce = agentstate.MaxSteeringForce

	return p
}

func (agent *Agent) GetState(swarmstate *state.ServerState) state.AgentState {
	return swarmstate.Agents[agent.Id]
}

func (agent *Agent) SetState(swarmstate *state.ServerState, state state.AgentState) {
	swarmstate.Agents[agent.Id] = state
}
