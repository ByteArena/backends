package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kardianos/osext"
	"github.com/netgusto/bytearena/server"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils/vector"
)

type cmdenvironment struct {
	host     string
	port     int
	tps      int
	agents   int
	agentimp string
}

func getcmdenv() cmdenvironment {

	// Host

	host, exists := os.LookupEnv("HOST")
	if !exists || host == "" {
		panic("You must set a valid HOST environment variable")
	}

	// Port
	var port int
	portstr, exists := os.LookupEnv("PORT")
	if !exists {
		port = 8080
	} else {
		portbis, err := strconv.Atoi(portstr)
		if err != nil {
			portbis = 8080
		}

		port = portbis
	}

	// Number of agents
	var nbagents int
	nbagentsstr, exists := os.LookupEnv("AGENTS")
	if !exists {
		nbagents = 2
	} else {
		nbagentsbis, err := strconv.Atoi(nbagentsstr)
		if err != nil {
			nbagentsbis = 2
		}
		nbagents = nbagentsbis
	}

	// Ticks per second
	var tps int
	tpsstr, exists := os.LookupEnv("TPS")
	if !exists {
		tps = 10
	} else {
		tpsbis, err := strconv.Atoi(tpsstr)
		if err != nil {
			tpsbis = 10
		}
		tps = tpsbis
	}

	// Agent implementation
	agentimp, exists := os.LookupEnv("AGENTIMP")
	if !exists {
		agentimp = "seeker"
	}

	return cmdenvironment{
		host:     host,
		port:     port,
		agents:   nbagents,
		agentimp: agentimp,
		tps:      tps,
	}
}

func main() {

	rand.Seed(time.Now().UnixNano())

	cmdenv := getcmdenv()

	exfolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	stopticking := make(chan bool)

	srv := server.NewServer(
		cmdenv.host,
		cmdenv.port,
		exfolder+"/../../agents/"+cmdenv.agentimp,
		cmdenv.agents,
		cmdenv.tps,
		stopticking,
	)

	// Creating obstacles

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(0, 0),
		vector.MakeVector2(1000, 0),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(1000, 0),
		vector.MakeVector2(1000, 600),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(1000, 600),
		vector.MakeVector2(0, 600),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(0, 600),
		vector.MakeVector2(0, 0),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(100, 100),
		vector.MakeVector2(900, 100),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(900, 100),
		vector.MakeVector2(900, 500),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(900, 500),
		vector.MakeVector2(500, 500),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(100, 500),
		vector.MakeVector2(100, 100),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(300, 300),
		vector.MakeVector2(300, 500),
	))

	srv.SetObstacle(state.MakeObstacle(
		vector.MakeVector2(700, 200),
		vector.MakeVector2(500, 400),
	))

	for i := 0; i < cmdenv.agents; i++ {
		go srv.Spawnagent()
	}

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		stopticking <- true
		srv.TearDown()
		os.Exit(1)
	}()

	go visualization(srv, cmdenv.host, cmdenv.port+1)

	srv.Listen()
}
