package deathmatch

import (
	"strconv"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

func (deathmatch *DeathmatchGame) NewEntityBallisticProjectile(ownerid ecs.EntityID, position vector.Vector2, velocity vector.Vector2) *ecs.Entity {

	projectile := deathmatch.manager.NewEntity()

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Type = box2d.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	bodydef.Position.Set(position.GetX(), position.GetY())
	bodydef.LinearVelocity = box2d.MakeB2Vec2(velocity.GetX(), velocity.GetY())

	body := deathmatch.PhysicalWorld.CreateBody(&bodydef)
	body.SetLinearDamping(0.0) // no aerodynamic drag

	shape := box2d.MakeB2CircleShape()
	shape.SetRadius(0.3)

	fixturedef := box2d.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixtureFromDef(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(
		types.PhysicalBodyDescriptorType.Projectile,
		strconv.Itoa(int(projectile.GetID())),
	))
	body.SetBullet(true)

	return projectile.
		AddComponent(deathmatch.physicalBodyComponent, &PhysicalBody{
			body:               body,
			maxSpeed:           100,
			maxSteeringForce:   100,
			maxAngularVelocity: 10,
			dragForce:          0,
		}).
		AddComponent(deathmatch.renderComponent, &Render{
			type_:  "projectile",
			static: false,
		}).
		AddComponent(deathmatch.ttlComponent, &Ttl{60}).
		AddComponent(deathmatch.ownedComponent, &Owned{ownerid})
}
