package deathmatch

import (
	"github.com/bytearena/ecs"
)

func impactWithDamage(deathmatch *DeathmatchGame, qrHealth *ecs.QueryResult, qrImpactor *ecs.QueryResult, killed *[]*ecs.QueryResult) {

	healthAspect := qrHealth.Components[deathmatch.healthComponent].(*Health)
	impactorAspect := qrImpactor.Components[deathmatch.impactorComponent].(*Impactor)

	healthAspect.AddLife(-1 * impactorAspect.damage)
	if healthAspect.GetLife() <= 0 {
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
			lifecycleQr := deathmatch.getEntity(coll.entityIDA, deathmatch.lifecycleComponent)
			if lifecycleQr == nil {
				impactWithDamage(deathmatch, entityResultAHealth, entityResultBImpactor, &killed)
			} else {
				lifecycleAspect := lifecycleQr.Components[deathmatch.lifecycleComponent].(*Lifecycle)
				if !lifecycleAspect.locked {
					impactWithDamage(deathmatch, entityResultAHealth, entityResultBImpactor, &killed)
				}
			}
		}

		if entityResultBHealth != nil && entityResultAImpactor != nil {
			lifecycleQr := deathmatch.getEntity(coll.entityIDB, deathmatch.lifecycleComponent)
			if lifecycleQr == nil {
				impactWithDamage(deathmatch, entityResultBHealth, entityResultAImpactor, &killed)
			} else {
				lifecycleAspect := lifecycleQr.Components[deathmatch.lifecycleComponent].(*Lifecycle)
				if !lifecycleAspect.locked {
					impactWithDamage(deathmatch, entityResultBHealth, entityResultAImpactor, &killed)
				}
			}
		}
	}

	for _, qrKilled := range killed {
		lifecycleQr := deathmatch.getEntity(qrKilled.Entity.GetID(), deathmatch.lifecycleComponent)
		if lifecycleQr == nil {
			continue
		}

		lifecycleAspect := lifecycleQr.Components[deathmatch.lifecycleComponent].(*Lifecycle)
		lifecycleAspect.SetDeath(deathmatch.ticknum)
	}
}
