package agent

import (
	"github.com/bytearena/bytearena/server/protocol"
	"github.com/bytearena/bytearena/server/state"
	uuid "github.com/satori/go.uuid"
)

type Agent interface {
	GetId() uuid.UUID
	String() string
	GetPerception(serverstate *state.ServerState) state.Perception
	SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) // abstract method
}

type AgentImp struct {
	id uuid.UUID
}

func MakeAgentImp() AgentImp {
	return AgentImp{
		id: uuid.NewV4(), // random uuid
	}
}

func (agent AgentImp) GetId() uuid.UUID {
	return agent.id
}

func (agent AgentImp) String() string {
	return "<AgentImp(" + agent.GetId().String() + ")>"
}

func (agent AgentImp) GetPerception(serverstate *state.ServerState) state.Perception {
	p := state.Perception{}
	agentstate := serverstate.GetAgentState(agent.GetId())

	orientation := agentstate.Orientation

	p.Internal.Velocity = agentstate.Velocity.Clone().SetAngle(agentstate.Velocity.Angle() - orientation)
	p.Internal.Proprioception = agentstate.Radius
	p.Internal.Magnetoreception = orientation // l'angle d'orientation de l'agent par rapport au "Nord" de l'arène

	p.Specs.MaxSpeed = agentstate.MaxSpeed
	p.Specs.MaxSteeringForce = agentstate.MaxSteeringForce
	p.Specs.MaxAngularVelocity = agentstate.MaxAngularVelocity
	p.Specs.VisionRadius = agentstate.VisionRadius
	p.Specs.VisionAngle = agentstate.VisionAngle

	p.External.Vision = agent.computeAgentVision(serverstate, agentstate)

	return p
}

func (agent AgentImp) SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) {
	// I'm abstract, override me !
}
