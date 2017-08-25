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

type GameInterface interface {
	GetId() string
	GetName() string
	GetTps() int
	GetRunStatus() int
	GetLaunchedAt() string
	GetEndedAt() string
	GetContestants() []Contestant
	GetMapContainer() *mapcontainer.MapContainer
}

type GameImpGql struct {
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

func NewGameGql(game graphqltype.GameType) *GameImpGql {

	// TODO(jerome): parametrize this
	jsonsource, err := FetchUrl("https://bytearena.com/assets/bytearena/maps/deathmatch/desert/death-valley/map.json")
	utils.Check(err, "Could not fetch map")

	var mapContainer mapcontainer.MapContainer
	err = json.Unmarshal(jsonsource, &mapContainer)
	utils.Check(err, "Could not load map JSON")

	return &GameImpGql{
		mapContainer: &mapContainer,
		gqlgame:      game,
	}
}

func (a *GameImpGql) GetId() string {
	return a.gqlgame.Id
}

func (a *GameImpGql) GetName() string {
	return a.gqlgame.Arena.Name
}

func (a *GameImpGql) GetTps() int {
	return a.gqlgame.Tps
}

func (a *GameImpGql) GetRunStatus() int {
	return a.gqlgame.RunStatus
}

func (a *GameImpGql) GetLaunchedAt() string {
	return a.gqlgame.LaunchedAt
}

func (a *GameImpGql) GetEndedAt() string {
	return a.gqlgame.EndedAt
}

func (a *GameImpGql) GetContestants() []Contestant {
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

func (a *GameImpGql) GetMapContainer() *mapcontainer.MapContainer {
	return a.mapContainer
}
