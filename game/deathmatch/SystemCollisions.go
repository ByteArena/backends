package deathmatch

import (
	"github.com/bytearena/box2d"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

type collision struct {
	entityIDA         ecs.EntityID
	entityIDB         ecs.EntityID
	collidableAspectA *Collidable
	collidableAspectB *Collidable
	point             vector.Vector2
	normal            vector.Vector2
	toi               float64
	friction          float64
	restitution       float64
}

func systemCollisions(deathmatch *DeathmatchGame) []collision {

	collisions := make([]collision, 0)

	for _, coll := range deathmatch.collisionListener.PopCollisions() {

		A, ok := coll.GetFixtureA().GetBody().GetUserData().(commontypes.PhysicalBodyDescriptor)
		if !ok {
			continue
		}

		B, ok := coll.GetFixtureB().GetBody().GetUserData().(commontypes.PhysicalBodyDescriptor)
		if !ok {
			continue
		}

		worldManifold := box2d.MakeB2WorldManifold()
		coll.GetWorldManifold(&worldManifold)

		entityResultA := deathmatch.getEntity(A.ID, deathmatch.collidableComponent)
		entityResultB := deathmatch.getEntity(B.ID, deathmatch.collidableComponent)

		if entityResultA == nil || entityResultB == nil {
			// Should never happen; this case is filtered in deathmatch.collisionFilter
			continue
		}

		collidableAspectA := deathmatch.CastCollidable(entityResultA.Components[deathmatch.collidableComponent])
		collidableAspectB := deathmatch.CastCollidable(entityResultB.Components[deathmatch.collidableComponent])

		compiledCollision := collision{
			entityIDA:         A.ID,
			entityIDB:         B.ID,
			collidableAspectA: collidableAspectA,
			collidableAspectB: collidableAspectB,
			point:             vector.FromB2Vec2(worldManifold.Points[0]),
			normal:            vector.FromB2Vec2(worldManifold.Normal),
			toi:               coll.GetTOI(),
			friction:          coll.GetFriction(),
			restitution:       coll.GetRestitution(),
		}

		collisions = append(collisions, compiledCollision)

		collidableAspectA.CollisionScript(deathmatch, A.ID, B.ID, collidableAspectA, collidableAspectB, compiledCollision.point)
		collidableAspectB.CollisionScript(deathmatch, B.ID, A.ID, collidableAspectB, collidableAspectA, compiledCollision.point)
	}

	return collisions
}
