package types

type NICIface struct {
	Model string
}

type NICSocket struct {
	Connect string
}

type NICTap struct {
	Name   string
	Ifname string
	Script string
}

type NICUser struct{}
