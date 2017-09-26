package deathmatch

type Lifecycle struct {
	tickBirth int
	tickDeath int
	maxAge    int
	onDeath   func()
}

func (deathmatch DeathmatchGame) CastLifecycle(data interface{}) *Lifecycle {
	return data.(*Lifecycle)
}

func (lc *Lifecycle) SetMaxAge(maxAge int) *Lifecycle {
	lc.maxAge = maxAge
	return lc
}

func (lc Lifecycle) GetBirth() int {
	return lc.tickBirth
}

func (lc *Lifecycle) SetBirth(tick int) *Lifecycle {
	lc.tickDeath = tick
	return lc
}

func (lc Lifecycle) GetDeath() int {
	return lc.tickDeath
}

func (lc *Lifecycle) SetDeath(tick int) *Lifecycle {
	lc.tickDeath = tick
	return lc
}
