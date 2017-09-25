package deathmatch

type Impactor struct {
	damage float64
}

func (deathmatch DeathmatchGame) CastImpactor(data interface{}) *Impactor {
	return data.(*Impactor)
}

func (o Impactor) GetDamage() float64 {
	return o.damage
}
