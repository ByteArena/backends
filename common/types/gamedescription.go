package types

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type GameDescriptionInterface interface {
	GetId() string
	GetName() string
	GetTps() int
	GetRunStatus() int
	GetLaunchedAt() string
	GetEndedAt() string
	GetContestants() []Contestant
	GetMapContainer() *mapcontainer.MapContainer
}

type GameDescriptionGQL struct {
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

func NewGameDescriptionGQL(game graphqltype.GameType) *GameDescriptionGQL {

	// TODO(jerome): parametrize this
	jsonsource, err := utils.FetchUrl("https://static.bytearena.com/assets/bytearena/maps/deathmatch/desert/death-valley/map.json")
	utils.Check(err, "Could not fetch map")

	var mapContainer mapcontainer.MapContainer
	err = json.Unmarshal(jsonsource, &mapContainer)
	utils.Check(err, "Could not load map JSON")

	return &GameDescriptionGQL{
		mapContainer: &mapContainer,
		gqlgame:      game,
	}
}

func (a *GameDescriptionGQL) GetId() string {
	return a.gqlgame.Id
}

func (a *GameDescriptionGQL) GetName() string {
	return a.gqlgame.Arena.Name
}

func (a *GameDescriptionGQL) GetTps() int {
	return a.gqlgame.Tps
}

func (a *GameDescriptionGQL) GetRunStatus() int {
	return a.gqlgame.RunStatus
}

func (a *GameDescriptionGQL) GetLaunchedAt() string {
	return a.gqlgame.LaunchedAt
}

func (a *GameDescriptionGQL) GetEndedAt() string {
	return a.gqlgame.EndedAt
}

func (a *GameDescriptionGQL) GetContestants() []Contestant {

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

func (a *GameDescriptionGQL) GetMapContainer() *mapcontainer.MapContainer {
	return a.mapContainer
}
