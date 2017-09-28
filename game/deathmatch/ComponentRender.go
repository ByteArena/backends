package deathmatch

type Render struct {
	type_  string
	static bool
}

func (r Render) GetType() string {
	return r.type_
}
