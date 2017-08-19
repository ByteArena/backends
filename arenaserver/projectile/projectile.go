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
	TTL            int
	AgentEmitterId uuid.UUID
}

func NewBallisticProjectile() *BallisticProjectile {
	return &BallisticProjectile{
		Id:    uuid.NewV4(), // random uuid
		TTL:   50,
		Speed: 6,
	}
}

func (p *BallisticProjectile) Update() {
	p.TTL--
	p.Position = p.Position.Add(p.Velocity)
}
