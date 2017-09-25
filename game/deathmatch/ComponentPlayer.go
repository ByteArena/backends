package deathmatch

type Player struct{}

func (deathmatch DeathmatchGame) CastPlayer(data interface{}) *Player {
	return data.(*Player)
}
