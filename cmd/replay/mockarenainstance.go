package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bytearena/core/common/types"
	"github.com/bytearena/core/common/types/mapcontainer"
	"github.com/bytearena/core/common/utils"
)

type MockGame struct {
	tps          int
	contestants  []types.Contestant
	mapContainer *mapcontainer.MapContainer
}

func NewMockGame(tps int) *MockGame {

	filepath := "../../maps/trainer-map.json"
	jsonsource, err := os.Open(filepath)
	if err != nil {
		utils.Debug("replay", "Error opening file: "+err.Error())
		os.Exit(1)
	}

	defer jsonsource.Close()

	bjsonmap, _ := ioutil.ReadAll(jsonsource)

	var mapContainer mapcontainer.MapContainer
	if err := json.Unmarshal(bjsonmap, &mapContainer); err != nil {
		utils.Debug("replay", "Could not load map JSON")
		os.Exit(1)
	}

	return &MockGame{
		tps:          tps,
		contestants:  make([]types.Contestant, 0),
		mapContainer: &mapContainer,
	}
}

func (game *MockGame) GetId() string {
	return "1"
}

func (game *MockGame) GetName() string {
	return "Replay game"
}

func (game *MockGame) GetTps() int {
	return game.tps
}

func (game *MockGame) GetRunStatus() int {
	return types.GameRunStatus.Running
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
		Username:      "replay-user",
		AgentName:     "Replay of " + agentimage,
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
