package deathmatch

import (
	"strconv"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils/number"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

func (deathmatch *DeathmatchGame) NewEntityAgent(position vector.Vector2) *ecs.Entity {

	agent := deathmatch.manager.NewEntity()

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Position.Set(position.GetX(), position.GetY())
	bodydef.Type = box2d.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	body := deathmatch.PhysicalWorld.CreateBody(&bodydef)

	shape := box2d.MakeB2CircleShape()
	shape.SetRadius(0.5)

	fixturedef := box2d.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixtureFromDef(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(
		types.PhysicalBodyDescriptorType.Agent,
		strconv.Itoa(int(agent.GetID())),
	))
	body.SetBullet(true)
	//body.SetLinearDamping(agentstate.DragForce * float64(s.tickspersec)) // aerodynamic drag

	return agent.
		AddComponent(deathmatch.physicalBodyComponent, &PhysicalBody{
			body:               body,
			maxSpeed:           0.75,
			maxSteeringForce:   0.12,
			maxAngularVelocity: number.DegreeToRadian(9),
			dragForce:          0.015,
		}).
		AddComponent(deathmatch.perceptionComponent, &Perception{
			visionAngle:  number.DegreeToRadian(180),
			visionRadius: 100,
		}).
		AddComponent(deathmatch.healthComponent, &Health{}).
		AddComponent(deathmatch.playerComponent, &Player{}).
		AddComponent(deathmatch.renderComponent, &Render{
			type_:  "agent",
			static: false,
		})
}
