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

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/arenaserver/container"
	"github.com/bytearena/bytearena/arenatrainer"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/protocol"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/vizserver"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {

	rand.Seed(time.Now().UnixNano())
	log.Println("Byte Arena Trainer v0.1")

	tickspersec := flag.Int("tps", 10, "Number of ticks per second")
	host := flag.String("host", "", "IP serving the trainer; required")
	port := flag.Int("port", 8080, "Port serving the trainer")

	var agentimages arrayFlags
	flag.Var(&agentimages, "agent", "Agent image in docker; example netgusto/meatgrinder")

	flag.Parse()

	if *host == "" {
		ip, err := utils.GetCurrentIP()
		utils.Check(err, "Could not determine host IP; you can specify using the `--host` flag.")
		*host = ip
	}

	if len(agentimages) == 0 {
		panic("Please, specify at least one agent image using --agent")
	}

	arena := NewMockArenaInstance(*tickspersec)
	for _, contestant := range agentimages {
		arena.AddContestant(contestant)
	}

	// Make message broker client
	brokerclient, err := arenatrainer.NewMemoryMessageClient()
	utils.Check(err, "ERROR: Could not connect to messagebroker")

	srv := arenaserver.NewServer(*host, *port, container.MakeLocalContainerOrchestrator(), arena)

	for _, contestant := range arena.GetContestants() {
		var image string

		if contestant.AgentRegistry == "" {
			image = contestant.AgentImage
		} else {
			image = contestant.AgentRegistry + "/" + contestant.AgentImage
		}

		srv.RegisterAgent(image)
	}

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		srv.Stop()
	}()

	go protocol.StreamState(srv, brokerclient)

	brokerclient.Subscribe("viz", "message", func(msg mq.BrokerMessage) {
		notify.PostTimeout("viz:message", string(msg.Data), time.Millisecond) // string because received as string from MQ, and no need to manipulate it on our side
	})

	go func(arenainstance arenaserver.ArenaInstance) {
		webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"

		vizservice := vizserver.NewVizService("0.0.0.0:"+strconv.Itoa(*port+1), webclientpath, func() ([]arenaserver.ArenaInstance, error) {
			res := make([]arenaserver.ArenaInstance, 1)
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
