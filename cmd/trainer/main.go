package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/common/messagebroker"
	commonprotocol "github.com/bytearena/bytearena/common/protocol"
	commonutils "github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/server"
	"github.com/bytearena/bytearena/server/state"
	"github.com/bytearena/bytearena/utils"
	"github.com/bytearena/bytearena/utils/vector"
	"github.com/bytearena/bytearena/vizserver"
	uuid "github.com/satori/go.uuid"
)

func main() {

	rand.Seed(time.Now().UnixNano())
	log.Println("Byte Arena Trainer v0.1")

	tickspersec := flag.Int("tps", 10, "Number of ticks per second")
	host := flag.String("host", "", "IP serving the trainer; required")
	port := flag.Int("port", 8080, "Port serving the trainer")

	flag.Parse()

	if *host == "" {
		ip, err := commonutils.GetCurrentIP()
		utils.Check(err, "Could not determine host IP; you can specify using the `--host` flag.")
		*host = ip
	}

	arena := MockArenaInstance{
		tps:           *tickspersec,
		agentregistry: "registry.bytearena.com",
		agentimage:    "xtuc/test",
	}

	// Make message broker client
	brokerclient, err := NewMemoryMessageClient()
	utils.Check(err, "ERROR: Could not connect to messagebroker")

	srv := server.NewServer(*host, *port, arena)

	for _, contestant := range arena.GetContestants() {
		srv.RegisterAgent(contestant.AgentRegistry + "/" + contestant.AgentImage)
	}

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		srv.Stop()
	}()

	go commonprotocol.StreamState(srv, brokerclient)

	brokerclient.Subscribe("viz", "message", func(msg messagebroker.BrokerMessage) {
		notify.PostTimeout("viz:message", string(msg.Data), time.Millisecond)
	})

	go func(arenainstance server.ArenaInstance) {
		webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"

		vizservice := vizserver.NewVizService("0.0.0.0:"+strconv.Itoa(*port+1), webclientpath, func() ([]server.ArenaInstance, error) {
			res := make([]server.ArenaInstance, 1)
			res[0] = arenainstance
			return res, nil
		})

		if err := vizservice.ListenAndServe(); err != nil {
			log.Panicln("Could not start viz service")
		}
	}(arena)

	<-srv.Start()
	srv.TearDown()
}

type MockArenaInstance struct {
	tps           int
	agentregistry string
	agentimage    string
}

func (ins MockArenaInstance) Setup(srv *server.Server) {
	srv.SetObstacle(state.Obstacle{
		Id: uuid.NewV4(),
		A:  vector.MakeVector2(0, 0),
		B:  vector.MakeVector2(1000, 0),
	})

	srv.SetObstacle(state.Obstacle{
		Id: uuid.NewV4(),
		A:  vector.MakeVector2(1000, 0),
		B:  vector.MakeVector2(1000, 1000),
	})

	srv.SetObstacle(state.Obstacle{
		Id: uuid.NewV4(),
		A:  vector.MakeVector2(1000, 1000),
		B:  vector.MakeVector2(0, 1000),
	})

	srv.SetObstacle(state.Obstacle{
		Id: uuid.NewV4(),
		A:  vector.MakeVector2(0, 1000),
		B:  vector.MakeVector2(0, 0),
	})
}

func (ins MockArenaInstance) GetId() string {
	return "1"
}

func (ins MockArenaInstance) GetName() string {
	return "Trainer instance"
}

func (ins MockArenaInstance) GetTps() int {
	return ins.tps
}

func (ins MockArenaInstance) GetSurface() server.PixelSurface {
	return server.PixelSurface{
		Width:  1000,
		Height: 1000,
	}
}

func (ins MockArenaInstance) GetContestants() []server.Contestant {
	res := make([]server.Contestant, 1)
	res[0] = server.Contestant{
		Id:            "1",
		Username:      "trainer-user",
		AgentName:     "Trainee",
		AgentRegistry: ins.agentregistry,
		AgentImage:    ins.agentimage,
	}

	return res
}
