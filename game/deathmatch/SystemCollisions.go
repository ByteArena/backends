package deathmatch

import (
	"strconv"

	"github.com/bytearena/box2d"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

func systemCollisions(deathmatch *DeathmatchGame) {
	for _, collision := range deathmatch.collisionListener.PopCollisions() {

		descriptorCollider, ok := collision.GetFixtureA().GetBody().GetUserData().(commontypes.PhysicalBodyDescriptor)
		if !ok {
			continue
		}

		descriptorCollidee, ok := collision.GetFixtureB().GetBody().GetUserData().(commontypes.PhysicalBodyDescriptor)
		if !ok {
			continue
		}

		if descriptorCollider.Type == commontypes.PhysicalBodyDescriptorType.Projectile {
			// on impacte le collider
			id, _ := strconv.Atoi(descriptorCollider.ID)
			entityid := ecs.EntityID(id)
			entityresult := deathmatch.getEntity(entityid, ecs.BuildTag(
				deathmatch.ttlComponent,
				deathmatch.playerComponent,
			))
			if entityresult == nil {
				continue
			}

			worldManifold := box2d.MakeB2WorldManifold()
			collision.GetWorldManifold(&worldManifold)

			ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent])
			physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

			ttlAspect.SetValue(1)

			physicalAspect.
				SetVelocity(vector.MakeNullVector2()).
				SetPosition(vector.FromB2Vec2(worldManifold.Points[0]))
		}

		if descriptorCollidee.Type == commontypes.PhysicalBodyDescriptorType.Projectile {
			// on impacte le collider
			id, _ := strconv.Atoi(descriptorCollidee.ID)
			entityid := ecs.EntityID(id)
			entityresult := deathmatch.getEntity(entityid, ecs.BuildTag(
				deathmatch.ttlComponent,
				deathmatch.playerComponent,
			))
			if entityresult == nil {
				continue
			}

			worldManifold := box2d.MakeB2WorldManifold()
			collision.GetWorldManifold(&worldManifold)

			ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent])
			physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

			ttlAspect.SetValue(1)

			physicalAspect.
				SetVelocity(vector.MakeNullVector2()).
				SetPosition(vector.FromB2Vec2(worldManifold.Points[0]))
		}
	}
}
