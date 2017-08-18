package arenaserver

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type Game interface {
	//Setup(srv *Server)
	GetId() string
	GetName() string
	GetTps() int
	GetContestants() []Contestant
	GetMapContainer() *mapcontainer.MapContainer
}

type GameGql struct {
	gqlgame      graphqltype.GameType
	mapContainer *mapcontainer.MapContainer
}

func FetchUrl(url string) ([]byte, error) {
	resp, err := http.Get(url)

	if err != nil && resp.StatusCode != 200 {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	return body, nil
}

func NewGameGql(game graphqltype.GameType) *GameGql {

	// filepath := "../../maps/trainer-map.json"
	// jsonsource, err := os.Open(filepath)
	// if err != nil {
	// 	log.Panicln("Error opening file:", err)
	// }

	// defer jsonsource.Close()

	jsonsource, err := FetchUrl("https://bytearena.com/assets/bytearena/maps/deathmatch/desert/death-valley/map.json")
	utils.Check(err, "Could not fetch map")

	var mapContainer mapcontainer.MapContainer
	if err := json.Unmarshal(jsonsource, &mapContainer); err != nil {
		log.Panicln("Could not load map JSON")
	}

	return &GameGql{
		mapContainer: &mapContainer,
		gqlgame:      game,
	}
}

func (a *GameGql) GetId() string {
	return a.gqlgame.Id
}

func (a *GameGql) GetName() string {
	return a.gqlgame.Arena.Name
}

func (a *GameGql) GetTps() int {
	return a.gqlgame.Tps
}

func (a *GameGql) GetContestants() []Contestant {
	log.Println(a.gqlgame.Contestants)
	res := make([]Contestant, len(a.gqlgame.Contestants))
	for index, contestant := range a.gqlgame.Contestants {
		res[index] = Contestant{
			Username:      contestant.Agent.Owner.Username,
			AgentName:     contestant.Agent.Name,
			AgentImage:    contestant.Agent.Image.Name + ":" + contestant.Agent.Image.Tag,
			AgentRegistry: contestant.Agent.Image.Registry,
		}
	}

	return res
}

// func (a *GameGql) Setup(srv *Server) {
// 	for _, obstacle := range a.gqlgame.Arena.Obstacles {
// 		srv.SetObstacle(state.MakeObstacle(
// 			vector.MakeVector2(obstacle.A.X, obstacle.A.Y),
// 			vector.MakeVector2(obstacle.B.X, obstacle.B.Y),
// 		))
// 	}
// }

func (a *GameGql) GetMapContainer() *mapcontainer.MapContainer {
	return a.mapContainer
}
