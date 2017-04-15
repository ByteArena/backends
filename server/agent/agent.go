package agent

import (
	"github.com/netgusto/bytearena/server/protocol"
	"github.com/netgusto/bytearena/server/state"
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
	//tickedchan chan utils.Tickturn
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

	p.Internal.Velocity = agentstate.Velocity.Clone()
	p.Internal.Proprioception = agentstate.Radius

	// l'angle d'orientation de l'agent par rapport au "Nord" de l'arène
	p.Internal.Magnetoreception = agentstate.Orientation

	p.Specs.MaxSpeed = agentstate.MaxSpeed
	p.Specs.MaxSteeringForce = agentstate.MaxSteeringForce
	p.Specs.MaxAngularVelocity = agentstate.MaxAngularVelocity

	// On calcule la perception Vision de l'agent
	serverstate.Agentsmutex.Lock()
	radiussq := agentstate.VisionRadius * agentstate.VisionRadius
	for otheragentid, otheragentstate := range serverstate.Agents {

		if otheragentid == agent.GetId() {
			continue
		}

		centervec := otheragentstate.Position.Sub(agentstate.Position)
		distsq := centervec.MagSq()
		if distsq <= radiussq {
			// Il faut aligner l'angle du vecteur sur le heading courant de l'agent
			centervec = centervec.SetAngle(centervec.Angle() - agentstate.Orientation)
			visionitem := state.PerceptionVisionItem{
				Center:   centervec,
				Radius:   otheragentstate.Radius,
				Velocity: otheragentstate.Velocity.Clone().SetAngle(otheragentstate.Velocity.Angle() - agentstate.Orientation),
				Tag:      otheragentstate.Tag,
			}

			p.External.Vision = append(p.External.Vision, visionitem)
		}
	}
	serverstate.Agentsmutex.Unlock()

	return p
}

func (agent AgentImp) SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) {
	// I'm abstract, override me !
}
