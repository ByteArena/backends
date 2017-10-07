package types

type NICIface struct {
	Model string
}

type NICSocket struct {
	Connect string
}

type NICTap struct {
	Ifname string
}

type NICUser struct {
	DHCPStart string
	Net       string
}

type QMPServer struct {
	Protocol string
	Addr     string
}

type NICBridge struct {
	Bridge string
	MAC    string
}
