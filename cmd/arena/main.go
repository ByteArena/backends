package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kardianos/osext"
	"github.com/netgusto/bytearena/agents/attractor"
	"github.com/netgusto/bytearena/server"
	"github.com/netgusto/bytearena/server/protocol"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils"
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

	// Creating attractor as an agent
	agentstate := state.MakeAgentState()
	agentstate.Tag = "attractor"
	agentstate.Position = utils.MakeVector2(400, 300)
	agentstate.Radius = 16
	srv.RegisterAgent(attractoragent.MakeAttractorAgent(), agentstate)

	filename := os.Args[1]
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Panicln(err)
	}

	var config protocol.FileConfigWrapper

	if err := json.Unmarshal(data, &config); err != nil {
		log.Panicln(err)
	}

	for _, agentconfig := range config.Agents {
		go srv.Spawnagent(agentconfig)
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
