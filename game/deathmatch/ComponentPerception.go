package deathmatch

type Perception struct {
	visionAngle  float64 // expressed in rad
	visionRadius float64 // expressed in rad
	perception   []byte
}

func (p Perception) GetVisionAngle() float64 {
	return p.visionAngle
}

func (p Perception) GetVisionRadius() float64 {
	return p.visionRadius
}

func (p *Perception) SetPerception(perception []byte) *Perception {
	p.perception = perception
	return p
}

func (p Perception) GetPerception() []byte {
	return p.perception
}
