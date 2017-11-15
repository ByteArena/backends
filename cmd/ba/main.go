package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/urfave/cli"

	"github.com/bytearena/bytearena/ba/action/generate"
	"github.com/bytearena/bytearena/ba/action/train"
	trainutils "github.com/bytearena/bytearena/ba/utils"
	bettererrors "github.com/xtuc/better-errors"

	mapcmd "github.com/bytearena/bytearena/ba/action/map"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	app := makeapp()
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
						NewFromString("Failed to execute command").
						SetContext("command", "generate").
						With(err)

					trainutils.FailWith(berror)
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
				cli.StringFlag{Name: "map", Value: "viz-island", Usage: "Name of the map used by the trainer"},
				cli.BoolFlag{Name: "no-browser", Usage: "Disable automatic browser opening at start"},
				cli.BoolFlag{Name: "debug", Usage: "Enable debug logging"},
				cli.BoolFlag{Name: "profile", Usage: "Enable execution profiling"},
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
				train.TrainAction(tps, host, port, nobrowser, recordFile, agents, isDebug, mapName, shouldProfile)
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
					Usage: "Update or fetch the trainer map",
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
			},
		},
	}

	return app
}
