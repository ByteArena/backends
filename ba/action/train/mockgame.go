package train

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/common/mappack"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"

	bettererrors "github.com/xtuc/better-errors"
)

type MockGame struct {
	tps          int
	contestants  []types.Contestant
	mapContainer *mapcontainer.MapContainer
}

func NewMockGame(tps int, mapbundle *mappack.MappackInMemoryArchive) (*MockGame, error) {

	jsonsource, err := mapbundle.Open("map.json")
	if err != nil {
		return nil, bettererrors.
			New("Could not find map.json inside the map bundle").
			With(bettererrors.NewFromErr(err))
	}

	var mapContainer mapcontainer.MapContainer
	if err := json.Unmarshal(jsonsource, &mapContainer); err != nil {

		return nil, bettererrors.
			New("map.json exists inside the map bundle, but is not valid.").
			With(bettererrors.NewFromErr(err))
	}

	return &MockGame{
		tps:          tps,
		contestants:  make([]types.Contestant, 0),
		mapContainer: &mapContainer,
	}, nil
}

func (game *MockGame) GetId() string {
	return "1"
}

func (game *MockGame) GetName() string {
	return "Trainer game"
}

func (game *MockGame) GetTps() int {
	return game.tps
}

func (game *MockGame) GetRunStatus() int {
	return gqltypes.GameRunStatus.Running
}

func (game *MockGame) GetLaunchedAt() string {
	return time.Now().Format("2006-01-02T15:04:05-0700")
}

func (game *MockGame) GetEndedAt() string {
	return ""
}

func (game *MockGame) AddContestant(agentimage string) {

	parts := strings.Split(agentimage, "/")
	var registry string
	var imagename string

	if len(parts) == 3 {
		registry = parts[0]
		imagename = strings.Join(parts[1:], "/")
	} else {
		registry = ""
		imagename = agentimage
	}

	game.contestants = append(game.contestants, types.Contestant{
		Id:            strconv.Itoa(len(game.contestants) + 1),
		Username:      "trainer-user",
		AgentName:     "Trainee " + agentimage,
		AgentRegistry: registry,
		AgentImage:    imagename,
	})
}

func (game *MockGame) GetContestants() []types.Contestant {
	return game.contestants
}

func (game *MockGame) GetMapContainer() *mapcontainer.MapContainer {
	return game.mapContainer
}
