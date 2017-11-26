package deathmatch

type stats struct {

	// Distance travelled by the agent in meters since the beginning of the game
	distanceTravelled float64

	nbBeenFragged uint
	nbHasFragged  uint

	nbBeenHit uint
	nbHasHit  uint
}

type Player struct {
	Name string

	// Populated by systemScore
	Score int

	Stats stats
}
