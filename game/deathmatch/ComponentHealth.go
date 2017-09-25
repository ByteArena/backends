package deathmatch

type Health struct {
	maxLife         float64 // Const
	life            float64 // Current life level
	deathScriptFunc func()

	// maxShield           float64 // Const
	// shield              float64 // Current shield level
	// shieldReplenishRate float64 // Const; Shield regained every tick
}

func NewHealth(maxlife float64) *Health {
	return &Health{
		maxLife: maxlife, // Const
		life:    maxlife, // Current life level

		// MaxShield:           1000, // Const
		// Shield:              1000, // Current shield level
		// ShieldReplenishRate: 10,   // Const; Shield regained every tick

		deathScriptFunc: nil,
	}
}

func (deathmatch DeathmatchGame) CastHealth(data interface{}) *Health {
	return data.(*Health)
}

func (health Health) GetMaxLife() float64 {
	return health.maxLife
}

func (health Health) GetLife() float64 {
	return health.life
}

func (health *Health) SetDeathScript(f func()) *Health {
	health.deathScriptFunc = f
	return health
}

func (health *Health) SetLife(life float64) {
	if life < 0 {
		life = 0
	}

	if life > health.maxLife {
		life = health.maxLife
	}

	health.life = life
}

func (health *Health) AddLife(life float64) {
	health.SetLife(life + health.GetLife())
}

func (health *Health) DeathScript() {
	if health.deathScriptFunc != nil {
		health.deathScriptFunc()
	}
}
