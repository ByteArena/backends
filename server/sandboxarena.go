package server

import (
	"github.com/bytearena/bytearena/server/state"
	"github.com/bytearena/bytearena/utils/vector"
)

type SandboxArena struct {
	specs ArenaSpecs
}

func NewSandboxArena() *SandboxArena {
	return &SandboxArena{
		specs: ArenaSpecs{
			Id:   "sandboxarena",
			Name: "Sandbox Arena",
			Surface: PixelSurface{
				Width:  800,
				Height: 600,
			},
		},
	}
}

func (a *SandboxArena) GetSpecs() ArenaSpecs {
	return a.specs
}

func (a *SandboxArena) Setup(srv *Server) {

	specs := a.GetSpecs()
	arenawidth := specs.Surface.Width.Pixels()
	arenaheight := specs.Surface.Height.Pixels()
	corridorbreadth := 100.0

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(0, 0),
		vector.MakeVector2(arenawidth, 0),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(arenawidth, 0),
		vector.MakeVector2(arenawidth, arenaheight),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(arenawidth, arenaheight),
		vector.MakeVector2(0, arenaheight),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(0, arenaheight),
		vector.MakeVector2(0, 0),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(corridorbreadth, corridorbreadth),
		vector.MakeVector2(arenawidth-corridorbreadth, corridorbreadth),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(arenawidth-corridorbreadth, corridorbreadth),
		vector.MakeVector2(arenawidth-corridorbreadth, arenaheight-corridorbreadth),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(arenawidth-corridorbreadth, arenaheight-corridorbreadth),
		vector.MakeVector2(arenawidth/2, arenaheight-corridorbreadth),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(corridorbreadth, arenaheight-corridorbreadth),
		vector.MakeVector2(corridorbreadth, corridorbreadth),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(corridorbreadth*3, corridorbreadth*3),
		vector.MakeVector2(corridorbreadth*3, arenaheight-corridorbreadth),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(arenawidth-corridorbreadth*3, corridorbreadth*2),
		vector.MakeVector2(arenawidth/2, arenaheight-corridorbreadth*1.5),
	))
}
