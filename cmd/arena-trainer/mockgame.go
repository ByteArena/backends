package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
)

type MockGame struct {
	tps          int
	contestants  []arenaserver.Contestant
	mapContainer *mapcontainer.MapContainer
}

func NewMockGame(tps int) *MockGame {

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

	return &MockGame{
		tps:          tps,
		contestants:  make([]arenaserver.Contestant, 0),
		mapContainer: &mapContainer,
	}
}

func (ins *MockGame) GetId() string {
	return "1"
}

func (ins *MockGame) GetName() string {
	return "Trainer game"
}

func (ins *MockGame) GetTps() int {
	return ins.tps
}

func (ins *MockGame) AddContestant(agentimage string) {

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

	ins.contestants = append(ins.contestants, arenaserver.Contestant{
		Id:            strconv.Itoa(len(ins.contestants) + 1),
		Username:      "trainer-user",
		AgentName:     "Trainee " + agentimage,
		AgentRegistry: registry,
		AgentImage:    imagename,
	})
}

func (ins *MockGame) GetContestants() []arenaserver.Contestant {
	return ins.contestants
}

func (ins *MockGame) GetMapContainer() *mapcontainer.MapContainer {
	return ins.mapContainer
}

/*
func (ins *MockGame) Setup(srv *arenaserver.Server) {

	mapcontainer := ins.GetMapContainer()
	for _, ground := range mapcontainer.Data.Grounds {
		for _, polygon := range ground.Outline {
			for i := 0; i < len(polygon.Points)-1; i++ {
				a := polygon.Points[i]
				b := polygon.Points[i+1]
				srv.SetObstacle(state.Obstacle{
					Id: uuid.NewV4(),
					A:  vector.MakeVector2(a.X*50, a.Y*50),
					B:  vector.MakeVector2(b.X*50, b.Y*50),
				})
			}
		}
	}
}
*/