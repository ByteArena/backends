package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/skratchdot/open-golang/open"

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
	bettererrors "github.com/xtuc/better-errors"
	bettererrorstree "github.com/xtuc/better-errors/printer/tree"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func debug(str string) {
	fmt.Println(str)
}

func failWith(err error) {
	if bettererrors.IsBetterError(err) {

		msg := bettererrorstree.PrintChain(err.(*bettererrors.Chain))

		urlOptions := url.Values{}
		urlOptions.Set("body", msg)

		fmt.Println("")
		fmt.Println("=== ")
		fmt.Println("=== ❌ an error occurred.")
		fmt.Println("===")
		fmt.Println("=== Please report this error here: https://github.com/ByteArena/trainer/issues/new?" + urlOptions.Encode())
		fmt.Println("=== We will fix it as soon as possible.")
		fmt.Println("===")
		fmt.Println("")

		fmt.Print(msg)

		os.Exit(1)
	} else {
		panic(err)
	}
}

func runPreflightChecks() {
	ensureDockerIsAvailable()
}

var (
	tickspersec      = flag.Int("tps", 10, "Number of ticks per second")
	host             = flag.String("host", "", "IP serving the trainer; required")
	port             = flag.Int("port", 8080, "Port serving the trainer")
	recordFile       = flag.String("record-file", "", "Destination file for recording the game")
	doNotOpenBrowser = flag.Bool("no-browser", false, "Disable automatic browser opening at start")
)

func main() {
	rand.Seed(time.Now().UnixNano())

	var agentimages arrayFlags
	flag.Var(&agentimages, "agent", "Agent image in docker; example netgusto/meatgrinder")

	flag.Parse()

	if *host == "" {
		ip, err := utils.GetCurrentIP()
		utils.Check(err, "Could not determine host IP; you can specify using the `--host` flag.")
		*host = ip
	}

	if len(agentimages) == 0 {
		fmt.Println("Please, specify at least one agent image using --agent")
		flag.Usage()
		os.Exit(1)
	}

	runPreflightChecks()

	// Make sure map exists locally and is update to date.
	mapManifest, errManifest := downloadAndGetManifest()
	if errManifest != nil {
		failWith(errManifest)
	}

	if isMapLocally() {
		mapChecksum, err := getLocalMapChecksum()
		if err != nil {
			failWith(err)
		}

		if mapChecksum != mapManifest.Md5 {
			debug("The map is outdated, downloading the new version...")

			err := downloadMap(mapManifest)

			if err != nil {
				failWith(err)
			}
		}
	} else {
		debug("Map doesn't exists locally, downloading...")

		err := downloadMap(mapManifest)

		if err != nil {
			failWith(err)
		}
	}

	gamedescription := NewMockGame(*tickspersec)
	for _, contestant := range agentimages {
		gamedescription.AddContestant(contestant)
	}

	// Make message broker client
	brokerclient, err := NewMemoryMessageClient()
	utils.Check(err, "ERROR: Could not connect to messagebroker")

	game := deathmatch.NewDeathmatchGame(gamedescription)

	srv := arenaserver.NewServer(*host, *port, container.MakeLocalContainerOrchestrator(*host), gamedescription, game, "", brokerclient)

	// consume server events
	go func() {
		events := srv.Events()

		for {
			msg := <-events

			switch t := msg.(type) {
			case arenaserver.EventStatusGameUpdate:
				fmt.Println(t.Status)

			case arenaserver.EventAgentLog:
				fmt.Println("agent", t.Value)
			case arenaserver.EventLog:
				fmt.Println("log", t.Value)

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
		debug("RECEIVED SHUTDOWN SIGNAL; closing.")
		srv.Stop()

		<-time.After(10 * time.Second)
		os.Exit(1)
	}()

	go common.StreamState(srv, brokerclient, "trainer")

	var recorder recording.RecorderInterface = recording.MakeEmptyRecorder()
	if *recordFile != "" {
		recorder = recording.MakeSingleArenaRecorder(*recordFile)
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

	mappack, errMappack := mappack.UnzipAndGetHandles(getMapLocation())

	if errMappack != nil {
		failWith(errMappack)
	}

	webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"
	vizservice := vizserver.NewVizService(
		"0.0.0.0:"+strconv.Itoa(*port+1),
		webclientpath,
		"training-dojo",
		func() ([]*types.VizGame, error) { return vizgames, nil },
		recorder,
		mappack,
	)

	vizservice.Start()

	serverChan, startErr := srv.Start()

	if startErr != nil {
		srv.Stop()
		failWith(startErr)
	}

	url := "http://localhost:" + strconv.Itoa(*port+1) + "/arena/1"

	if !*doNotOpenBrowser {
		open.Run(url)
	}
	fmt.Println("\033[0;34m\nGame running at " + url + "\033[0m\n")

	<-serverChan

	srv.TearDown()

	recorder.Close(gamedescription.GetId())
	recorder.Stop()

	vizservice.Stop()
}
