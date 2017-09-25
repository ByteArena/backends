package deathmatch

import "github.com/bytearena/ecs"

func systemTtl(deathmatch *DeathmatchGame) {
	entitiesToRemove := make([]*ecs.Entity, 0)

	for _, entityresult := range deathmatch.ttlView.Get() {
		ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent])
		if ttlAspect.Decrement(1) < 0 {
			entitiesToRemove = append(entitiesToRemove, entityresult.Entity)
		}
	}

	deathmatch.manager.DisposeEntities(entitiesToRemove...)
}
