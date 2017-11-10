package deathmatch

import (
	"github.com/bytearena/ecs"
)

func systemHealth(deathmatch *DeathmatchGame, collisions []collision) {

	killed := make([]*ecs.QueryResult, 0)

	for _, coll := range collisions {
		entityResultAImpactor := deathmatch.getEntity(coll.entityIDA, deathmatch.impactorComponent)
		entityResultAHealth := deathmatch.getEntity(coll.entityIDA, deathmatch.healthComponent)

		entityResultBImpactor := deathmatch.getEntity(coll.entityIDB, deathmatch.impactorComponent)
		entityResultBHealth := deathmatch.getEntity(coll.entityIDB, deathmatch.healthComponent)

		if entityResultAImpactor != nil && entityResultAHealth != nil && entityResultBImpactor != nil {
			impactIfPossible(deathmatch, entityResultAImpactor, entityResultAHealth, entityResultBImpactor, &killed)
		}

		if entityResultBImpactor != nil && entityResultBHealth != nil && entityResultAImpactor != nil {
			impactIfPossible(deathmatch, entityResultBImpactor, entityResultBHealth, entityResultAImpactor, &killed)
		}
	}

	for _, qrKilled := range killed {
		lifecycleQr := deathmatch.getEntity(qrKilled.Entity.GetID(), deathmatch.lifecycleComponent)
		if lifecycleQr == nil {
			continue
		}

		lifecycleAspect := lifecycleQr.Components[deathmatch.lifecycleComponent].(*Lifecycle)
		lifecycleAspect.SetDeath(deathmatch.ticknum)

		// TODO: LOG EVENT HASFRAGGED on impactor
		// TODO: LOG EVENT DEATH on impactee; OR MAYBE IN lifecycleAspect.SetDeath ?
		deathmatch.log.AddEntry(MakeLogEntryOfType(EVENT_PROJECTILE_KILLED_ENTITY, qrKilled.Entity))
	}
}

func impactIfPossible(deathmatch *DeathmatchGame, impactee *ecs.QueryResult, impacteeHealth *ecs.QueryResult, impactor *ecs.QueryResult, killed *[]*ecs.QueryResult) {

	lifecycleQr := deathmatch.getEntity(impactee.Entity.ID, deathmatch.lifecycleComponent)
	if lifecycleQr == nil {

		// no lifecycle on impactee; cannot be locked, impacting !
		impactWithDamage(deathmatch, impacteeHealth, impactor, killed)
	} else {

		// There's a lifecycle on impactee; check if entity is locked
		lifecycleAspect := lifecycleQr.Components[deathmatch.lifecycleComponent].(*Lifecycle)

		if !lifecycleAspect.locked {
			// impactee not be locked, impacting !
			impactWithDamage(deathmatch, impacteeHealth, impactor, killed)
		}
	}
}

func impactWithDamage(deathmatch *DeathmatchGame, qrHealth *ecs.QueryResult, qrImpactor *ecs.QueryResult, killed *[]*ecs.QueryResult) {

	healthAspect := qrHealth.Components[deathmatch.healthComponent].(*Health)
	impactorAspect := qrImpactor.Components[deathmatch.impactorComponent].(*Impactor)

	// TODO: LOG EVENT TOOKHIT on impactee, and HASHIT on impactor

	healthAspect.AddLife(-1 * impactorAspect.damage)
	if healthAspect.GetLife() <= 0 {
		healthAspect.SetLife(0)
		*killed = append(*killed, qrHealth)
	}
}
