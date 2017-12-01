package main

import (
	"github.com/bytearena/bytearena/common/types"
	structdoc "github.com/xtuc/go-structdoc"
)

// COPIED

type Specs struct {
	MaxSpeed           float64     `json:"maxspeed"`
	MaxSteeringForce   float64     `json:"maxsteeringforce"`
	MaxAngularVelocity float64     `json:"maxangularvelocity"`
	VisionRadius       float64     `json:"visionradius"`
	VisionAngle        types.Angle `json:"visionangle"`

	BodyRadius float64 `json:"bodyradius"`

	MaxShootEnergy    float64 `json:"maxshootenergy"`
	ShootRecoveryRate float64 `json:"shootrecoveryrate"`

	Gear map[string]GearSpecs
}

type GearSpecs struct {
	Genre string
	Kind  string
	Specs interface{}
}

type GunSpecs struct {
	ShootCost        float64 `json:"shootcost"`
	ShootCooldown    int     `json:"shootcooldown"`
	ProjectileSpeed  float64 `json:"projectilespeed"`
	ProjectileDamage float64 `json:"projectiledamage"`
	ProjectileRange  float64 `json:"projectilerange"`
}

// COPIED

var (
	handshakeRuntimeTypes = map[string]string{
		"unknown":   "Object",
		"Angle":     "Number (radian)",
		"GearSpecs": "Object",
		"float64":   "Number",
		"int":       "Number",
		"string":    "string",
	}
)

func handshakeNormalizeTypeName(t string) string {

	if t == "types.Angle" {

		return "Angle"
	} else if t == "map[string]main.GearSpecs" {

		return "GearSpecs"
	} else if t == "vector.Vector2" {

		return "Vector2"
	} else if t == "interface {}" {

		return "unknown"
	} else {

		return t
	}
}

func main() {
	generator := structdoc.MakeGenerator(handshakeNormalizeTypeName, handshakeRuntimeTypes)

	generator.GeneratorFor(Specs{})
	generator.GeneratorFor(GearSpecs{})
	generator.GeneratorFor(GunSpecs{})
}
