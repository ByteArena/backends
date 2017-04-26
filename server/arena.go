package server

type Arena interface {
	Setup(srv *Server)
	GetSpecs() ArenaSpecs
}

type PixelSize2D struct {
	Width  float64
	Height float64
}

type ArenaSpecs struct {
	Name         string
	DimensionsPx PixelSize2D
}
