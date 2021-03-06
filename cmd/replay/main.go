package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/skratchdot/open-golang/open"

	"github.com/bytearena/core/common"
	"github.com/bytearena/core/common/recording"
	"github.com/bytearena/core/common/utils"
	"github.com/bytearena/core/common/visualization"
	"github.com/bytearena/core/common/visualization/types"
)

func main() {
	filename := flag.String("file", "", "Filename")
	port := flag.Int("viz-port", 8080, "Specifiy the port of the visualization server")

	flag.Parse()

	utils.Assert(*filename != "", "--file must be set")

	game := NewMockGame(10)

	vizserver := NewVizService(*port, game, *filename)
	// Below line is used to serve assets locally

	vizserver.Start()

	url := "http://localhost:" + strconv.Itoa(*port) + "/record/1"

	fmt.Println("\033[0;34m\nReplay ready; open " + url + " in your browser.\033[0m\n")
	open.Run(url)

	<-common.SignalHandler()

	utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")
	vizserver.Stop()
}

func sendMapToViz(msg string, debug bool, UUID string) {
	if debug {
		utils.Debug("viz-server", "read buffer of length: "+strconv.Itoa(len(msg)))
	}

	notify.PostTimeout("viz:map:"+UUID, msg, time.Millisecond)
}

func NewVizService(port int, game *MockGame, recordFile string) *visualization.VizService {

	recordStore := recording.NewSingleFileRecordStore(recordFile)

	// TODO(jerome|sven): refac webclient path / serving
	webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"

	vizgames := make([]*types.VizGame, 1)
	vizgames[0] = types.NewVizGame(game)

	vizservice := visualization.NewVizService("0.0.0.0:"+strconv.Itoa(port), webclientpath, func() ([]*types.VizGame, error) {
		return vizgames, nil
	}, recordStore)

	return vizservice
}
