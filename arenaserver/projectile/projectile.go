package projectile

import (
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type BallisticProjectile struct {
	Id             uuid.UUID
	Position       vector.Vector2
	Velocity       vector.Vector2
	Speed          float64
	Radius         float64
	TTL            int
	AgentEmitterId uuid.UUID
	JustFired      bool
}

func NewBallisticProjectile() *BallisticProjectile {
	return &BallisticProjectile{
		Id:     uuid.NewV4(), // random uuid
		TTL:    50,
		Speed:  6,
		Radius: 0.3,
	}
}

func (p *BallisticProjectile) Update() {
	if p.JustFired {
		p.JustFired = false
	} else {
		p.TTL--
		p.Position = p.Position.Add(p.Velocity)
	}
}
