package deathmatch

import (
	"log"

	"github.com/bytearena/ecs"
)

func impactWithDamage(deathmatch *DeathmatchGame, qrHealth *ecs.QueryResult, qrImpactor *ecs.QueryResult, killed *[]*Health) {
	log.Println("AGENT TOOK A HIT !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	healthAspect := deathmatch.CastHealth(qrHealth.Components[deathmatch.healthComponent])
	impactorAspect := deathmatch.CastImpactor(qrImpactor.Components[deathmatch.impactorComponent])

	healthAspect.AddLife(-1 * impactorAspect.damage)
	if healthAspect.GetLife() <= 0 {
		*killed = append(*killed, healthAspect)
	}
}

func systemHealth(deathmatch *DeathmatchGame, collisions []collision) {

	killed := make([]*Health, 0)
	for _, coll := range collisions {
		entityResultAImpactor := deathmatch.getEntity(coll.entityIDA, deathmatch.impactorComponent)
		entityResultAHealth := deathmatch.getEntity(coll.entityIDA, deathmatch.healthComponent)

		entityResultBImpactor := deathmatch.getEntity(coll.entityIDB, deathmatch.impactorComponent)
		entityResultBHealth := deathmatch.getEntity(coll.entityIDB, deathmatch.healthComponent)

		if entityResultAHealth != nil && entityResultBImpactor != nil {
			impactWithDamage(deathmatch, entityResultAHealth, entityResultBImpactor, &killed)
		}

		if entityResultBHealth != nil && entityResultAImpactor != nil {
			impactWithDamage(deathmatch, entityResultBHealth, entityResultAImpactor, &killed)
		}
	}

	for _, killedHealthAspect := range killed {
		killedHealthAspect.DeathScript()
	}

	// for _, coll := range collisions {
	// 	if descriptorCollider.Type == commontypes.PhysicalBodyDescriptorType.Projectile {
	// 		// on impacte le collider
	// 		entityid := descriptorCollider.ID
	// 		entityresult := deathmatch.getEntity(entityid, ecs.BuildTag(
	// 			deathmatch.ttlComponent,
	// 			deathmatch.playerComponent,
	// 		))
	// 		if entityresult == nil {
	// 			continue
	// 		}

	// 		worldManifold := box2d.MakeB2WorldManifold()
	// 		coll.GetWorldManifold(&worldManifold)

	// 		collisions = append(collisions, collision{
	// 			entityA:
	// 		})

	// 		// ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent])
	// 		// physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

	// 		// ttlAspect.SetValue(1)

	// 		// physicalAspect.
	// 		// 	SetVelocity(vector.MakeNullVector2()).
	// 		// 	SetPosition(vector.FromB2Vec2(worldManifold.Points[0]))
	// 	}

	// 	if descriptorCollidee.Type == commontypes.PhysicalBodyDescriptorType.Projectile {
	// 		// on impacte le collider
	// 		entityid := descriptorCollidee.ID
	// 		entityresult := deathmatch.getEntity(entityid, ecs.BuildTag(
	// 			deathmatch.ttlComponent,
	// 			deathmatch.playerComponent,
	// 		))
	// 		if entityresult == nil {
	// 			continue
	// 		}

	// 		worldManifold := box2d.MakeB2WorldManifold()
	// 		coll.GetWorldManifold(&worldManifold)

	// 		// ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent])
	// 		// physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

	// 		// ttlAspect.SetValue(1)

	// 		// physicalAspect.
	// 		// 	SetVelocity(vector.MakeNullVector2()).
	// 		// 	SetPosition(vector.FromB2Vec2(worldManifold.Points[0]))
	// 	}
	// }
}
