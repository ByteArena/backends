package deathmatch

func systemLifecycle(deathmatch *DeathmatchGame) {
	for _, entityresult := range deathmatch.lifecycleView.Get() {
		lifecycleAspect := deathmatch.CastLifecycle(entityresult.Components[deathmatch.lifecycleComponent])
		if lifecycleAspect.maxAge > 0 && (deathmatch.ticknum-lifecycleAspect.tickBirth) > lifecycleAspect.maxAge {
			lifecycleAspect.SetDeath(deathmatch.ticknum)
		}
	}
}
