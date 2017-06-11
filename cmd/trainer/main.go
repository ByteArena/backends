package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonprotocol "github.com/bytearena/bytearena/common/protocol"
	"github.com/bytearena/bytearena/server"
)

func main() {

	rand.Seed(time.Now().UnixNano())
	log.Println("Byte Arena Trainer v0.1")

	tickspersec := flag.Int("tps", 10, "Number of ticks per second")
	host := flag.String("host", "", "IP serving the trainer; required")
	port := flag.Int("port", 8080, "Port serving the trainer")

	flag.Parse()

	if *host == "" {
		fmt.Println("-host is required")
		os.Exit(1)
	}

	arena := MockArenaInstance{
		tps:           *tickspersec,
		agentregistry: "registry.bytearena.com",
		agentimage:    "xtuc/test",
	}

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

	<-srv.Start()
	srv.TearDown()
}

type MockArenaInstance struct {
	tps           int
	agentregistry string
	agentimage    string
}

func (ins MockArenaInstance) Setup(srv *server.Server) {

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
