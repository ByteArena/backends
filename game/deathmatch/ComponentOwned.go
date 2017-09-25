package deathmatch

import "github.com/bytearena/ecs"

type Owned struct {
	owner ecs.EntityID
}

func (deathmatch DeathmatchGame) CastOwned(data interface{}) *Owned {
	return data.(*Owned)
}

func (o Owned) GetOwner() ecs.EntityID {
	return o.owner
}

func (o *Owned) SetOwner(owner ecs.EntityID) *Owned {
	o.owner = owner
	return o
}
