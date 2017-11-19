package train

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime/pprof"
	"strconv"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/arenaserver/container"
	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/mappack"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/game/deathmatch"
	"github.com/bytearena/bytearena/vizserver"
	"github.com/bytearena/bytearena/vizserver/types"
	"github.com/skratchdot/open-golang/open"

	mapcmd "github.com/bytearena/bytearena/ba/action/map"
	bettererrors "github.com/xtuc/better-errors"
)

const (
	TIME_BEFORE_FORCE_QUIT = 10 * time.Second
)

func TrainAction(tps int, host string, port int, nobrowser bool, recordFile string, agentimages []string, isDebug bool, mapName string, shouldProfile, dumpRaw bool) {

	if shouldProfile {
		f, err := os.Create("./cpu.prof")
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	shutdownChan := make(chan bool)
	debug := func(str string) {}

	if isDebug {
		debug = func(str string) {
			fmt.Printf("[debug] %s\n", str)
		}
	}

	if host == "" {
		ip, err := utils.GetCurrentIP()
		utils.Check(err, "Could not determine host IP; you can specify using the `--host` flag.")
		host = ip
	}

	if len(agentimages) == 0 {
		fmt.Println("Please, specify at least one agent image using --agent")
		flag.Usage()
		os.Exit(1)
	}

	runPreflightChecks()

	mappack, errMappack := mappack.UnzipAndGetHandles(mapcmd.GetMapLocation(mapName))
	if errMappack != nil {
		utils.FailWith(errMappack)
	}

	gamedescription, err := NewMockGame(tps, mappack)
	if err != nil {
		utils.FailWith(err)
	}
	for _, contestant := range agentimages {
		gamedescription.AddContestant(contestant)
	}

	// Make message broker client
	brokerclient, err := NewMemoryMessageClient()
	utils.Check(err, "ERROR: Could not connect to messagebroker")

	game := deathmatch.NewDeathmatchGame(gamedescription)

	srv := arenaserver.NewServer(host, port, container.MakeLocalContainerOrchestrator(host), gamedescription, game, "", brokerclient)

	// consume server events
	go func() {
		events := srv.Events()

		for {
			msg := <-events

			switch t := msg.(type) {
			case arenaserver.EventStatusGameUpdate:
				fmt.Println("[game]", t.Status)

			case arenaserver.EventAgentLog:
				fmt.Println("[agent]", t.Value)

			case arenaserver.EventLog:
				fmt.Println("[log]", t.Value)

			case arenaserver.EventDebug:
				debug(t.Value)

			case arenaserver.EventError:
				utils.FailWith(t.Err)

			case arenaserver.EventWarn:
				utils.WarnWith(t.Err)

			case arenaserver.EventRawComm:
				if dumpRaw {
					fmt.Printf("[agent] %s", t.Value)
				}

			case arenaserver.EventClose:
				return

			default:
				msg := fmt.Sprintf("Unsupported message of type %s", reflect.TypeOf(msg))
				panic(msg)
			}
		}
	}()

	go func() {
		utils.LogFn = func(service, message string) {
			fmt.Println(message)
		}
	}()

	for _, contestant := range gamedescription.GetContestants() {
		var image string

		if contestant.AgentRegistry == "" {
			image = contestant.AgentImage
		} else {
			image = contestant.AgentRegistry + "/" + contestant.AgentImage
		}

		srv.RegisterAgent(image, image)
	}

	// handling signals
	go func() {
		<-common.SignalHandler()
		shutdownChan <- true
	}()

	go common.StreamState(srv, brokerclient, "trainer")

	var recorder recording.RecorderInterface = recording.MakeEmptyRecorder()
	if recordFile != "" {
		recorder = recording.MakeSingleArenaRecorder(recordFile)
	}

	recorder.RecordMetadata(gamedescription.GetId(), gamedescription.GetMapContainer())

	brokerclient.Subscribe("viz", "message", func(msg mq.BrokerMessage) {
		gameID := gamedescription.GetId()

		recorder.Record(gameID, string(msg.Data))
		notify.PostTimeout("viz:message:"+gameID, string(msg.Data), time.Millisecond)
	})

	// TODO(jerome): refac webclient path / serving

	vizgames := make([]*types.VizGame, 1)
	vizgames[0] = types.NewVizGame(gamedescription)

	webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"
	vizservice := vizserver.NewVizService(
		"0.0.0.0:"+strconv.Itoa(port+1),
		webclientpath,
		mapName,
		func() ([]*types.VizGame, error) { return vizgames, nil },
		recorder,
		mappack,
	)

	vizservice.Start()

	serverChan, startErr := srv.Start()

	if startErr != nil {
		utils.FailWith(startErr)
	}

	url := "http://localhost:" + strconv.Itoa(port+1) + "/arena/1"

	if !nobrowser {
		open.Run(url)
	}

	fmt.Println("\033[0;34m\nGame running at " + url + "\033[0m\n")

	// Wait until someone asks for shutdown
	select {
	case <-serverChan:
	case <-shutdownChan:
	}

	// Force quit if the programs didn't exit
	go func() {
		<-time.After(TIME_BEFORE_FORCE_QUIT)

		berror := bettererrors.New("Forced shutdown")

		utils.FailWith(berror)
	}()

	debug("Shutdown...")

	srv.Stop()

	recorder.Close(gamedescription.GetId())
	recorder.Stop()

	vizservice.Stop()
}
