package deathmatch

import (
	"fmt"

	"github.com/bytearena/box2d"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/ecs"
)

func (deathmatch *DeathmatchGame) NewEntityGround(polygon mapcontainer.MapPolygon, name string) *ecs.Entity {
	return newEntityGroundOrObstacle(deathmatch, polygon, commontypes.PhysicalBodyDescriptorType.Ground, name).
		AddComponent(deathmatch.collidableComponent, NewCollidable(
			CollisionGroup.Ground,
			utils.BuildTag(
				CollisionGroup.Agent,
			),
		))
}

func (deathmatch *DeathmatchGame) NewEntityObstacle(polygon mapcontainer.MapPolygon, name string) *ecs.Entity {
	return newEntityGroundOrObstacle(deathmatch, polygon, commontypes.PhysicalBodyDescriptorType.Obstacle, name).
		AddComponent(deathmatch.collidableComponent, NewCollidable(
			CollisionGroup.Obstacle,
			utils.BuildTag(
				CollisionGroup.Agent,
				CollisionGroup.Projectile,
			),
		))
}

func newEntityGroundOrObstacle(deathmatch *DeathmatchGame, polygon mapcontainer.MapPolygon, obstacletype string, name string) *ecs.Entity {

	obstacle := deathmatch.manager.NewEntity()

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Type = box2d.B2BodyType.B2_staticBody

	body := deathmatch.PhysicalWorld.CreateBody(&bodydef)
	vertices := make([]box2d.B2Vec2, len(polygon.Points)) // -1: avoid last point because the last point of the loop should not be repeated

	for i := 0; i < len(polygon.Points); i++ {
		vertices[i].Set(polygon.Points[i].GetX(), polygon.Points[i].GetY()*-1) // TODO(jerome): invert axes in transform, not here
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\n\nERROR - Obstacle or ground (type " + obstacletype + ") " + name + " is not valid; perhaps some vertices are duplicated?\n\n")
			panic(r)
		}
	}()

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
