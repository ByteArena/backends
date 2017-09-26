package deathmatch

import "github.com/bytearena/ecs"

func systemDeath(deathmatch *DeathmatchGame) {

	entitiesToRemove := make([]*ecs.Entity, 0)

	for _, entityresult := range deathmatch.lifecycleView.Get() {
		lifecycleAspect := deathmatch.CastLifecycle(entityresult.Components[deathmatch.lifecycleComponent])
		if lifecycleAspect.tickDeath == deathmatch.ticknum {
			if lifecycleAspect.onDeath != nil {
				lifecycleAspect.onDeath()
			} else {
				entitiesToRemove = append(entitiesToRemove, entityresult.Entity)
			}
		}
	}

	if len(entitiesToRemove) > 0 {
		deathmatch.manager.DisposeEntities(entitiesToRemove...)
	}
}
