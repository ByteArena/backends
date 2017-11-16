package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/urfave/cli"

	"github.com/bytearena/bytearena/ba/action/generate"
	"github.com/bytearena/bytearena/ba/action/train"
	"github.com/bytearena/bytearena/common/utils"
	bettererrors "github.com/xtuc/better-errors"

	mapcmd "github.com/bytearena/bytearena/ba/action/map"
)

func main() {
	defer func() {
		if data := recover(); data != nil {

			if err, ok := data.(error); ok {

				berror := bettererrors.NewFromErr(err)
				utils.FailWith(berror)
			} else if str, ok := data.(string); ok {

				berror := bettererrors.New(str)
				utils.FailWith(berror)
			} else {

				panic(data)
			}
		}
	}()

	rand.Seed(time.Now().UnixNano())

	app := makeapp()
	app.Version = utils.GetVersion()
	app.Run(os.Args)
}

func makeapp() *cli.App {
	app := cli.NewApp()
	app.Description = "Byte Arena cli tool"
	app.Name = "Byte Arena cli tool"

	app.Commands = []cli.Command{
		{
			Name:    "generate",
			Aliases: []string{"gen"},
			Usage:   "Generate a boilerplate agent",
			Action: func(c *cli.Context) error {
				err := generate.Main(c.Args().Get(0))

				if err != nil {
					berror := bettererrors.
						New("Failed to execute command").
						SetContext("command", "generate").
						With(err)

					utils.FailWith(berror)
				}

				return nil
			},
		},
		{
			Name:    "train",
			Aliases: []string{"t"},
			Usage:   "Train your agent",
			Flags: []cli.Flag{
				cli.IntFlag{Name: "tps", Value: 10, Usage: "Number of ticks per second"},
				cli.StringFlag{Name: "host", Value: "", Usage: "IP serving the trainer; required"},
				cli.StringSliceFlag{Name: "agent", Usage: "Agent images"},
				cli.IntFlag{Name: "port", Value: 8080, Usage: "Port serving the trainer"},
				cli.StringFlag{Name: "record-file", Value: "", Usage: "Destination file for recording the game"},
				cli.StringFlag{Name: "map", Value: "island", Usage: "Name of the map used by the trainer"},
				cli.BoolFlag{Name: "no-browser", Usage: "Disable automatic browser opening at start"},
				cli.BoolFlag{Name: "debug", Usage: "Enable debug logging"},
				cli.BoolFlag{Name: "profile", Usage: "Enable execution profiling"},
				cli.BoolFlag{Name: "dump-raw-comm", Usage: "Dump all the communication between the agent and the server"},
			},
			Action: func(c *cli.Context) error {
				tps := c.Int("tps")
				host := c.String("host")
				agents := c.StringSlice("agent")
				port := c.Int("port")
				recordFile := c.String("record-file")
				mapName := c.String("map")
				nobrowser := c.Bool("no-browser")
				isDebug := c.Bool("debug")
				shouldProfile := c.Bool("profile")
				dumpRaw := c.Bool("dump-raw-comm")

				train.TrainAction(
					tps,
					host,
					port,
					nobrowser,
					recordFile,
					agents,
					isDebug,
					mapName,
					shouldProfile,
					dumpRaw,
				)

				return nil
			},
		},
		{
			Name:    "map",
			Aliases: []string{},
			Usage:   "Operations on map packs",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "debug", Usage: "Enable debug logging"},
			},
			Subcommands: []cli.Command{
				{
					Name:  "update",
					Usage: "Fetch the trainer maps if needed",
					Action: func(c *cli.Context) error {
						isDebug := c.Bool("debug")

						debug := func(str string) {}

						if isDebug {
							debug = func(str string) {
								fmt.Println(str)
							}
						}

						mapcmd.MapUpdateAction(debug)
						return nil
					},
				},
				{
					Name:  "list",
					Usage: "List the trainer maps locally available",
					Action: func(c *cli.Context) error {
						mapcmd.MapListAction()
						return nil
					},
				},
			},
		},
	}

	return app
}
