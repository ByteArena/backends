package deathmatch

type Ttl struct {
	ttl int
}

func (deathmatch DeathmatchGame) CastTtl(data interface{}) *Ttl {
	return data.(*Ttl)
}

func (t *Ttl) SetValue(ttl int) *Ttl {
	t.ttl = ttl
	return t
}

func (t *Ttl) Decrement(amount int) int {
	t.ttl -= amount
	return t.ttl
}

func (t *Ttl) Increment(amount int) int {
	t.ttl += amount
	return t.ttl
}

func (t Ttl) GetValue(ttl int) int {
	return t.ttl
}

func (t *Ttl) Step() *Ttl {
	t.ttl -= 1
	return t
}
