package state

import "github.com/bytearena/bytearena/common/utils/vector"

type PerceptionSpecs struct {
	// Weight int
	// statique
	// TBD
	MaxSpeed           float64 // max distance covered per turn
	MaxSteeringForce   float64 // max force applied when steering (ie, max magnitude of steering vector)
	MaxAngularVelocity float64
	VisionRadius       float64
	VisionAngle        float64
	DragForce          float64
}

type PerceptionVisionItem struct {
	Tag       string // should be an enum, but no enum in Go
	CloseEdge vector.Vector2
	Center    vector.Vector2
	FarEdge   vector.Vector2
	Velocity  vector.Vector2
}

type PerceptionExternal struct {
	Vision []PerceptionVisionItem // TBD
	Sound  []vector.Vector2       // tableau de vecteurs (volume et direction) dans un espace quantisé
	Touch  int                    // TBD; collisions ?
	Time   int                    // en ms depuis le début de la partie
	Radar  int                    // TBD; perception des obstacles ? position, vélocité, nature; position: segment 1d obstruant l'horizon 1D pour un monde 2D (à la Super hexagon) ?
	Xray   int                    // TBD; vision à travers les obstacles
}

type PerceptionInternal struct {
	Energy           float64        // niveau en millièmes; reconstitution automatique ?
	Proprioception   float64        // rayon de la surface occupée par le corps en rayon par rapport au centre géométrique
	Temperature      float64        // en degrés
	Balance          vector.Vector2 // vecteur de longeur 1 pointant depuis le centre de gravité vers la négative du vecteur gravité
	Velocity         vector.Vector2 // vecteur de force (direction, magnitude)
	Acceleration     vector.Vector2 // vecteur de force (direction, magnitude)
	Gravity          vector.Vector2 // vecteur de force (direction, magnitude)
	Damage           float64        // fiabilité générale en millièmes, fiabilité par système en millièmes
	Magnetoreception float64        // azimuth en degrés par rapport au "Nord" de l'arène
}

type PerceptionObjective struct {
	Attractor         vector.Vector2
	AttractorVelocity vector.Vector2
	// TBD
	// mission ?
	// sens de la course ?
	// port du flag ou non ?
	// position du flag ?
}

type Perception struct {
	Specs     PerceptionSpecs
	External  PerceptionExternal
	Internal  PerceptionInternal
	Objective PerceptionObjective
}
