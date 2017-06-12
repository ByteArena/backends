package server

import (
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/bytearena/server/state"

	"log"

	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
)

type ArenaInstance interface {
	Setup(srv *Server)
	GetId() string
	GetName() string
	GetTps() int
	GetSurface() PixelSurface
	GetContestants() []Contestant
}

type ArenaInstanceGql struct {
	gqlarenainstance graphqltype.ArenaInstanceType
}

func NewArenaInstanceGql(arenainstance graphqltype.ArenaInstanceType) *ArenaInstanceGql {
	return &ArenaInstanceGql{
		gqlarenainstance: arenainstance,
	}
}

func (a *ArenaInstanceGql) GetId() string {
	return a.gqlarenainstance.Id
}

func (a *ArenaInstanceGql) GetName() string {
	return a.gqlarenainstance.Arena.Name
}

func (a *ArenaInstanceGql) GetTps() int {
	return a.gqlarenainstance.Tps
}
func (a *ArenaInstanceGql) GetSurface() PixelSurface {
	return PixelSurface{
		Width:  PixelUnit(a.gqlarenainstance.Arena.Surface.Width),
		Height: PixelUnit(a.gqlarenainstance.Arena.Surface.Height),
	}
}

func (a *ArenaInstanceGql) GetContestants() []Contestant {
	log.Println(a.gqlarenainstance.Contestants)
	res := make([]Contestant, len(a.gqlarenainstance.Contestants))
	for index, contestant := range a.gqlarenainstance.Contestants {
		res[index] = Contestant{
			Username:      contestant.Agent.Owner.Username,
			AgentName:     contestant.Agent.Name,
			AgentImage:    contestant.Agent.Image.Name + ":" + contestant.Agent.Image.Tag,
			AgentRegistry: contestant.Agent.Image.Registry,
		}
	}

	return res
}

func (a *ArenaInstanceGql) Setup(srv *Server) {
	for _, obstacle := range a.gqlarenainstance.Arena.Obstacles {
		srv.SetObstacle(state.MakeObstacle(
			vector.MakeVector2(obstacle.A.X, obstacle.A.Y),
			vector.MakeVector2(obstacle.B.X, obstacle.B.Y),
		))
	}
}
