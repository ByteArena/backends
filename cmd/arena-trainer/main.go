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
	"github.com/urfave/cli"

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

func main() {
	rand.Seed(time.Now().UnixNano())

	app := makeapp()
	app.Run(os.Args)

}

func makeapp() *cli.App {
	app := cli.NewApp()

	app.Commands = []cli.Command{
		{
			Name:    "trainer",
			Aliases: []string{"t"},
			Usage:   "Train your agent",
			Flags: []cli.Flag{
				cli.IntFlag{Name: "tps", Value: 10, Usage: "Number of ticks per second"},
				cli.StringFlag{Name: "host", Value: "", Usage: "IP serving the trainer; required"},
				cli.StringSliceFlag{Name: "agent", Usage: "Agent images"},
				cli.IntFlag{Name: "port", Value: 8080, Usage: "Port serving the trainer"},
				cli.StringFlag{Name: "record-file", Value: "", Usage: "Destination file for recording the game"},
				cli.BoolFlag{Name: "no-browser", Usage: "Disable automatic browser opening at start"},
			},
			Action: func(c *cli.Context) error {
				nobrowser := c.Bool("no-browser")
				tps := c.Int("tps")
				host := c.String("host")
				port := c.Int("port")
				recordFile := c.String("record-file")
				agents := c.StringSlice("agent")
				trainAction(tps, host, port, nobrowser, recordFile, agents)
				return nil
			},
		},
		{
			Name:    "map",
			Aliases: []string{},
			Usage:   "Operations on map packs",
			Subcommands: []cli.Command{
				{
					Name:  "update",
					Usage: "Update or fetch the trainer map",
					Action: func(c *cli.Context) error {
						mapUpdateAction()
						return nil
					},
				},
			},
		},
	}

	return app
}

func trainAction(tps int, host string, port int, nobrowser bool, recordFile string, agentimages []string) {
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

	if isMapLocally() {
		// nothing to do
	} else {
		debug("Map doesn't exists locally, downloading...")

		// Make sure map exists locally and is update to date.
		mapManifest, errManifest := downloadAndGetManifest()
		if errManifest != nil {
			failWith(errManifest)
		}
		err := downloadMap(mapManifest)

		if err != nil {
			failWith(err)
		}
	}

	gamedescription, err := NewMockGame(tps)
	if err != nil {
		failWith(err)
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

	mappack, errMappack := mappack.UnzipAndGetHandles(getMapLocation())

	if errMappack != nil {
		failWith(errMappack)
	}

	webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"
	vizservice := vizserver.NewVizService(
		"0.0.0.0:"+strconv.Itoa(port+1),
		webclientpath,
		"viz-island",
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

	url := "http://localhost:" + strconv.Itoa(port+1) + "/arena/1"

	if !nobrowser {
		open.Run(url)
	}
	fmt.Println("\033[0;34m\nGame running at " + url + "\033[0m\n")

	<-serverChan

	srv.TearDown()

	recorder.Close(gamedescription.GetId())
	recorder.Stop()

	vizservice.Stop()

}

func mapUpdateAction() {
	mapChecksum, err := getLocalMapChecksum()
	if err != nil {
		// failWith(err)
		// Local map has never been downloaded
		fmt.Println("Map does not exist locally; will have to be fetched.")
	}

	fmt.Println("Downloading map manifest from " + MANIFEST_URL)

	mapManifest, errManifest := downloadAndGetManifest()
	if errManifest != nil {
		failWith(errManifest)
	}

	if mapChecksum != mapManifest.Md5 {
		debug("The map is outdated, downloading the new version...")

		err := downloadMap(mapManifest)

		if err != nil {
			failWith(err)
		}
	} else {
		debug("The map is already up to date!")
	}
}
