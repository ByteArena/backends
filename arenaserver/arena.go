package arenaserver

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
)

type ArenaInstance interface {
	//Setup(srv *Server)
	GetId() string
	GetName() string
	GetTps() int
	GetContestants() []Contestant
	GetMapContainer() *mapcontainer.MapContainer
}

type ArenaInstanceGql struct {
	gqlarenainstance graphqltype.ArenaInstanceType
	mapContainer     *mapcontainer.MapContainer
}

func NewArenaInstanceGql(arenainstance graphqltype.ArenaInstanceType) *ArenaInstanceGql {

	filepath := "../../maps/trainer-map.json"
	jsonsource, err := os.Open(filepath)
	if err != nil {
		log.Panicln("Error opening file:", err)
	}

	defer jsonsource.Close()

	bjsonmap, _ := ioutil.ReadAll(jsonsource)

	var mapContainer mapcontainer.MapContainer
	if err := json.Unmarshal(bjsonmap, &mapContainer); err != nil {
		log.Panicln("Could not load map JSON")
	}

	return &ArenaInstanceGql{
		mapContainer:     &mapContainer,
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

// func (a *ArenaInstanceGql) Setup(srv *Server) {
// 	for _, obstacle := range a.gqlarenainstance.Arena.Obstacles {
// 		srv.SetObstacle(state.MakeObstacle(
// 			vector.MakeVector2(obstacle.A.X, obstacle.A.Y),
// 			vector.MakeVector2(obstacle.B.X, obstacle.B.Y),
// 		))
// 	}
// }

func (a *ArenaInstanceGql) GetMapContainer() *mapcontainer.MapContainer {
	return a.mapContainer
}
