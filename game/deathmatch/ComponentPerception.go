package deathmatch

type Perception struct {
	visionAngle  float64 // expressed in rad
	visionRadius float64 // expressed in rad
}

func (p Perception) GetVisionAngle() float64 {
	return p.visionAngle
}

func (p Perception) GetVisionRadius() float64 {
	return p.visionRadius
}
