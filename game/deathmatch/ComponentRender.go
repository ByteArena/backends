package deathmatch

type Render struct {
	type_  string
	static bool
}

func (deathmatch DeathmatchGame) CastRender(data interface{}) *Render {
	return data.(*Render)
}

func (r Render) GetType() string {
	return r.type_
}
