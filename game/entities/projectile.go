package entities

import (
	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/common/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type BallisticProjectile struct {
	Id             uuid.UUID
	TTL            int
	AgentEmitterId uuid.UUID
	JustFired      bool
	PhysicalBody   *box2d.B2Body // replaces Radius, Mass, Position, Velocity, Orientation
}

func NewBallisticProjectile(Id uuid.UUID, body *box2d.B2Body) *BallisticProjectile {
	return &BallisticProjectile{
		Id:           Id, // random uuid
		PhysicalBody: body,
	}
}

func (p BallisticProjectile) GetPosition() vector.Vector2 {
	v := p.PhysicalBody.GetPosition()
	return vector.MakeVector2(v.X, v.Y)
}

func (p BallisticProjectile) GetVelocity() vector.Vector2 {
	v := p.PhysicalBody.GetLinearVelocity()
	return vector.MakeVector2(v.X, v.Y)
}

func (p BallisticProjectile) GetRadius() float64 {
	// FIXME(jerome): here we suppose that the agent is always a circle
	return p.PhysicalBody.GetFixtureList().GetShape().GetRadius()
}

func (p *BallisticProjectile) Update() {
	if p.JustFired {
		p.JustFired = false
	} else {
		p.TTL--
	}
}
