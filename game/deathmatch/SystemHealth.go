package deathmatch

import (
	"log"

	"github.com/bytearena/ecs"
)

func impactWithDamage(deathmatch *DeathmatchGame, qrHealth *ecs.QueryResult, qrImpactor *ecs.QueryResult, killed *[]*ecs.QueryResult) {
	log.Println("AGENT TOOK A HIT !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	healthAspect := deathmatch.CastHealth(qrHealth.Components[deathmatch.healthComponent])
	impactorAspect := deathmatch.CastImpactor(qrImpactor.Components[deathmatch.impactorComponent])

	healthAspect.AddLife(-1 * impactorAspect.damage * 1000)
	if healthAspect.GetLife() <= 0 {
		log.Println("AGENT KILLED !!!!!!!!!!!!!!!!!!!!!!!!")
		*killed = append(*killed, qrHealth)
	}
}

func systemHealth(deathmatch *DeathmatchGame, collisions []collision) {

	killed := make([]*ecs.QueryResult, 0)
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

	for _, qrKilled := range killed {
		healthAspect := deathmatch.CastHealth(qrKilled.Components[deathmatch.healthComponent])

		if healthAspect.DeathScript != nil {
			healthAspect.DeathScript()
		} else {
			lifecycleQr := deathmatch.getEntity(qrKilled.Entity.GetID(), deathmatch.lifecycleComponent)
			if lifecycleQr == nil {
				continue
			}

			lifecycleAspect := deathmatch.CastLifecycle(qrKilled.Components[deathmatch.lifecycleComponent])
			lifecycleAspect.SetDeath(deathmatch.ticknum)
		}
	}
}
