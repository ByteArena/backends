package main

import (
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/bytearena/game/deathmatch/mailboxmessages"
	structdoc "github.com/xtuc/go-structdoc"
)

// COPIED

type MessageWrapper struct {
	Subject string      `json:"subject"`
	Body    interface{} `json:"body"`
}

type VisionItem struct {
	Tag      string         `json:"tag"`
	NearEdge vector.Vector2 `json:"nearedge"`
	Center   vector.Vector2 `json:"center"`
	FarEdge  vector.Vector2 `json:"faredge"`
	Velocity vector.Vector2 `json:"velocity"`
}

type Perception struct {
	Score int `json:"score"`

	Energy        float64          `json:"energy"`   // niveau en millièmes; reconstitution automatique ?
	Velocity      vector.Vector2   `json:"velocity"` // vecteur de force (direction, magnitude)
	Azimuth       float64          `json:"azimuth"`  // azimuth en degrés par rapport au "Nord" de l'arène
	Vision        []VisionItem     `json:"vision"`
	ShootEnergy   float64          `json:"shootenergy"`
	ShootCooldown int              `json:"shootcooldown"`
	Messages      []MessageWrapper `json:"messages"`
}

// COPIED

var (
	perceptionRuntimeTypes = map[string]string{
		"unknown":   "Object",
		"Angle":     "Number (radian)",
		"GearSpecs": "Object",
		"float64":   "Number",
		"int":       "Number",
		"string":    "string",
	}
)

func perceptionNormalizeTypeName(t string) string {

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
	generator := structdoc.MakeGenerator(perceptionNormalizeTypeName, perceptionRuntimeTypes)

	generator.GeneratorFor(Perception{})
	generator.GeneratorFor(VisionItem{})
	generator.GeneratorFor(MessageWrapper{})

	generator.GeneratorFor(mailboxmessages.Score{})
	generator.GeneratorFor(mailboxmessages.Stats{})
	generator.GeneratorFor(mailboxmessages.YouAreRespawning{})
	generator.GeneratorFor(mailboxmessages.YouHaveBeenFragged{})
	generator.GeneratorFor(mailboxmessages.YouHaveBeenHit{})
	generator.GeneratorFor(mailboxmessages.YouHaveFragged{})
	generator.GeneratorFor(mailboxmessages.YouHaveHit{})
	generator.GeneratorFor(mailboxmessages.YouHaveRespawned{})
}
