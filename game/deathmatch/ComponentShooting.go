package deathmatch

import (
	"sync"

	"github.com/bytearena/bytearena/common/utils/vector"
)

type Shooting struct {
	pendingShots []vector.Vector2
	lock         *sync.RWMutex

	MaxShootEnergy           float64 // Const; When shooting, energy decreases
	ShootEnergy              float64 // Current energy level
	ShootEnergyReplenishRate float64 // Const; Energy regained every tick
	ShootEnergyCost          float64 // Const; Energy consumed by a shot
	ShootCooldown            int     // Const; number of ticks to wait between every shot
	LastShot                 int     // Number of ticks since last shot
}

func NewShooting() *Shooting {
	return &Shooting{
		lock: &sync.RWMutex{},

		MaxShootEnergy:           200, // Const; When shooting, energy decreases
		ShootEnergy:              200, // Current energy level
		ShootEnergyReplenishRate: 5,   // Const; Energy regained every tick
		ShootCooldown:            2,   // Const; number of ticks to wait between every shot
		ShootEnergyCost:          0,   // Const
		LastShot:                 0,   // Number of ticks since last shot; 0 => cannot shoot immediately, must wait for first cooldown
	}
}

func (deathmatch DeathmatchGame) CastShooting(data interface{}) *Shooting {
	return data.(*Shooting)
}

func (shooting *Shooting) PushShot(aiming vector.Vector2) {
	shooting.lock.Lock()
	shooting.pendingShots = append(shooting.pendingShots, aiming)
	shooting.lock.Unlock()
}

func (shooting *Shooting) PopPendingShots() []vector.Vector2 {
	shooting.lock.RLock()
	res := shooting.pendingShots
	shooting.pendingShots = make([]vector.Vector2, 0)
	shooting.lock.RUnlock()

	return res
}
