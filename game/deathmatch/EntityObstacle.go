package deathmatch

import (
	"github.com/bytearena/box2d"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/ecs"
)

func (deathmatch *DeathmatchGame) NewEntityGround(polygon mapcontainer.MapPolygon) *ecs.Entity {
	return newEntityGroundOrObstacle(deathmatch, polygon, commontypes.PhysicalBodyDescriptorType.Ground).
		AddComponent(deathmatch.collidableComponent, NewCollidable(
			CollisionGroup.Ground,
			utils.BuildTag(
				CollisionGroup.Agent,
			),
		))
}

func (deathmatch *DeathmatchGame) NewEntityObstacle(polygon mapcontainer.MapPolygon) *ecs.Entity {
	return newEntityGroundOrObstacle(deathmatch, polygon, commontypes.PhysicalBodyDescriptorType.Obstacle).
		AddComponent(deathmatch.collidableComponent, NewCollidable(
			CollisionGroup.Obstacle,
			utils.BuildTag(
				CollisionGroup.Agent,
				CollisionGroup.Projectile,
			),
		))
}

func newEntityGroundOrObstacle(deathmatch *DeathmatchGame, polygon mapcontainer.MapPolygon, obstacletype string) *ecs.Entity {

	obstacle := deathmatch.manager.NewEntity()

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Type = box2d.B2BodyType.B2_staticBody

	body := deathmatch.PhysicalWorld.CreateBody(&bodydef)
	vertices := make([]box2d.B2Vec2, len(polygon.Points)-1) // -1: avoid last point because the last point of the loop should not be repeated

	for i := 0; i < len(polygon.Points)-1; i++ {
		vertices[i].Set(polygon.Points[i].X, polygon.Points[i].Y)
	}

	shape := box2d.MakeB2ChainShape()
	shape.CreateLoop(vertices, len(vertices))
	body.CreateFixture(&shape, 0.0)
	body.SetUserData(commontypes.MakePhysicalBodyDescriptor(
		obstacletype,
		obstacle.GetID(),
	))

	return obstacle.
		AddComponent(deathmatch.physicalBodyComponent, &PhysicalBody{
			body:   body,
			static: true,
		})
}
