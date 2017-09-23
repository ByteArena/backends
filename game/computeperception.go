package game

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/ecs"
)

func (game *DeathmatchGame) ComputeAgentPerception(arenaMap *mapcontainer.MapContainer, entityid ecs.EntityID) protocol.AgentPerception {
	p := protocol.AgentPerception{}

	entityresult := game.GetEntity(entityid, ecs.BuildTag(
		game.physicalBodyComponent,
		game.perceptionComponent,
	))

	if entityresult == nil {
		return p
	}

	physicalAspect := game.CastPhysicalBody(entityresult.Components[game.physicalBodyComponent.GetID()])
	perceptionAspect := game.CastPerception(entityresult.Components[game.perceptionComponent.GetID()])

	orientation := physicalAspect.GetOrientation()
	velocity := physicalAspect.GetVelocity()
	radius := physicalAspect.GetRadius()

	p.Internal.Velocity = velocity.Clone().SetAngle(velocity.Angle() - orientation)
	p.Internal.Proprioception = radius
	p.Internal.Magnetoreception = orientation // l'angle d'orientation de l'agent par rapport au "Nord" de l'arène

	p.Specs.MaxSpeed = physicalAspect.GetMaxSpeed()
	p.Specs.MaxSteeringForce = physicalAspect.GetMaxSteeringForce()
	p.Specs.MaxAngularVelocity = physicalAspect.GetMaxAngularVelocity()
	p.Specs.DragForce = physicalAspect.GetDragForce()
	p.Specs.VisionRadius = perceptionAspect.GetVisionRadius()
	p.Specs.VisionAngle = perceptionAspect.GetVisionAngle()

	p.External.Vision = game.ComputeAgentVision(arenaMap, entityresult.Entity, physicalAspect, perceptionAspect)

	return p
}
