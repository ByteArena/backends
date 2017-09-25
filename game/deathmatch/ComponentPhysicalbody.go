package deathmatch

import (
	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/common/utils/vector"
)

func (deathmatch DeathmatchGame) CastPhysicalBody(data interface{}) *PhysicalBody {
	return data.(*PhysicalBody)
}

type PhysicalBody struct {
	body               *box2d.B2Body
	maxSpeed           float64 // expressed in m/tick
	maxSteeringForce   float64 // expressed in m/tick
	maxAngularVelocity float64 // expressed in rad/tick
	visionRadius       float64 // expressed in m
	visionAngle        float64 // expressed in rad
	dragForce          float64 // expressed in m/tick
}

func (p *PhysicalBody) GetBody() *box2d.B2Body {
	return p.body
}

func (p *PhysicalBody) SetBody(body *box2d.B2Body) *PhysicalBody {
	p.body = body
	return p
}

func (p PhysicalBody) GetPosition() vector.Vector2 {
	v := p.body.GetPosition()
	return vector.MakeVector2(v.X, v.Y)
}

func (p *PhysicalBody) SetPosition(v vector.Vector2) *PhysicalBody {
	p.body.SetTransform(v.ToB2Vec2(), p.GetOrientation())
	return p
}

func (p PhysicalBody) GetVelocity() vector.Vector2 {
	v := p.body.GetLinearVelocity()
	return vector.MakeVector2(v.X, v.Y)
}

func (p *PhysicalBody) SetVelocity(v vector.Vector2) *PhysicalBody {
	// FIXME(jerome): properly convert units from m/tick to m/s for Box2D
	p.body.SetLinearVelocity(v.Scale(20).ToB2Vec2())
	return p
}

func (p PhysicalBody) GetOrientation() float64 {
	return p.body.GetAngle()
}

func (p *PhysicalBody) SetOrientation(angle float64) *PhysicalBody {
	// Could also be implemented using torque; see http://www.iforce2d.net/b2dtut/rotate-to-angle
	p.body.SetTransform(p.body.GetPosition(), angle)
	return p
}

func (p PhysicalBody) GetRadius() float64 {
	// FIXME(jerome): here we suppose that the agent is always a circle
	return p.body.GetFixtureList().GetShape().GetRadius()
}

func (p PhysicalBody) GetMaxSpeed() float64 {
	return p.maxSpeed
}

func (p *PhysicalBody) SetMaxSpeed(maxSpeed float64) *PhysicalBody {
	p.maxSpeed = maxSpeed
	return p
}

func (p PhysicalBody) GetMaxSteeringForce() float64 {
	return p.maxSteeringForce
}

func (p *PhysicalBody) SetMaxSteeringForce(maxSteeringForce float64) *PhysicalBody {
	p.maxSteeringForce = maxSteeringForce
	return p
}

func (p PhysicalBody) GetMaxAngularVelocity() float64 {
	return p.maxAngularVelocity
}

func (p *PhysicalBody) SetMaxAngularVelocity(maxAngularVelocity float64) *PhysicalBody {
	p.maxAngularVelocity = maxAngularVelocity
	return p
}

func (p PhysicalBody) GetVisionRadius() float64 {
	return p.visionRadius
}

func (p *PhysicalBody) SetVisionRadius(visionRadius float64) *PhysicalBody {
	p.visionRadius = visionRadius
	return p
}

func (p PhysicalBody) GetVisionAngle() float64 {
	return p.visionAngle
}

func (p *PhysicalBody) SetVisionAngle(visionAngle float64) *PhysicalBody {
	p.visionAngle = visionAngle
	return p
}

func (p PhysicalBody) GetDragForce() float64 {
	return p.dragForce
}

func (p *PhysicalBody) SetDragForce(dragForce float64) *PhysicalBody {
	p.dragForce = dragForce
	return p
}
