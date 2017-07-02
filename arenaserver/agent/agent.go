package agent

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
	agentstate "github.com/bytearena/bytearena/arenaserver/state/agent"
	stateprotocol "github.com/bytearena/bytearena/arenaserver/state/protocol"
	serverstate "github.com/bytearena/bytearena/arenaserver/state/server"
	uuid "github.com/satori/go.uuid"
)

type Agent interface {
	GetId() uuid.UUID
	String() string
	GetPerception(serverstate *serverstate.ServerState) agentstate.Perception
	SetPerception(perception agentstate.Perception, comm protocol.AgentCommunicator, agentstate stateprotocol.AgentStateInterface) // abstract method
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

func (agent AgentImp) GetPerception(serverstate *serverstate.ServerState) agentstate.Perception {
	p := agentstate.Perception{}
	agstate := serverstate.GetAgentState(agent.GetId())

	orientation := agstate.GetOrientation()
	velocity := agstate.GetVelocity()
	radius := agstate.GetRadius()
	visionRadius := agstate.GetVisionRadius()
	visionAngle := agstate.GetVisionAngle()

	p.Internal.Velocity = velocity.Clone().SetAngle(velocity.Angle() - orientation)
	p.Internal.Proprioception = radius
	p.Internal.Magnetoreception = orientation // l'angle d'orientation de l'agent par rapport au "Nord" de l'arène

	p.Specs.VisionRadius = visionRadius
	p.Specs.VisionAngle = visionAngle

	// TODO(netgusto) : handle specifics of agent depending on drive mode (here, props for holonomic drive)
	// p.Specs.MaxSpeed = agstate.MaxSpeed
	// p.Specs.MaxSteeringForce = agstate.MaxSteeringForce
	// p.Specs.MaxAngularVelocity = agstate.MaxAngularVelocity

	p.External.Vision = agent.computeAgentVision(serverstate, agstate)

	return p
}

func (agent AgentImp) SetPerception(perception agentstate.Perception, comm protocol.AgentCommunicator, agentstate stateprotocol.AgentStateInterface) {
	// I'm abstract, override me !
}
