package types

type ContainerId string

func (c ContainerId) String() string {
	return string(c)
}
