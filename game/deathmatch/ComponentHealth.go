package deathmatch

type Health struct{}

func (deathmatch DeathmatchGame) GetHealth(data interface{}) *Health {
	return data.(*Health)
}
