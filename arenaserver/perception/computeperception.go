package perception

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/game/entities"
)

func ComputeAgentPerception(arenaMap *mapcontainer.MapContainer, serverstate *state.ServerState, agent entities.AgentInterface) protocol.AgentPerception {
	p := protocol.AgentPerception{}
	agentstate := serverstate.GetAgentState(agent.GetId())

	orientation := agentstate.GetOrientation()

	p.Internal.Velocity = agentstate.GetVelocity().Clone().SetAngle(agentstate.GetVelocity().Angle() - orientation)
	p.Internal.Proprioception = agentstate.GetRadius()
	p.Internal.Magnetoreception = orientation // l'angle d'orientation de l'agent par rapport au "Nord" de l'arène

	p.Specs.MaxSpeed = agentstate.MaxSpeed
	p.Specs.MaxSteeringForce = agentstate.MaxSteeringForce
	p.Specs.MaxAngularVelocity = agentstate.MaxAngularVelocity
	p.Specs.VisionRadius = agentstate.VisionRadius
	p.Specs.VisionAngle = agentstate.VisionAngle
	p.Specs.DragForce = agentstate.DragForce

	p.External.Vision = ComputeAgentVision(arenaMap, serverstate, agent)

	return p
}
