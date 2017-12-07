package replay

import (
	"fmt"
	"strconv"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/skratchdot/open-golang/open"

	mapcmd "github.com/bytearena/bytearena/ba/action/map"
	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/mappack"
	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/vizserver"
	"github.com/bytearena/bytearena/vizserver/types"
)

func Main(filename string, port int) {
	game := NewMockGame(10)

	vizserver := NewVizService(port, game, filename)
	// Below line is used to serve assets locally
	// TODO(jerome): find a way to bundle the trainer with the assets
	//vizserver.SetPathToAssets("/Users/jerome/Code/other/assets/")

	vizserver.Start()

	url := "http://localhost:" + strconv.Itoa(port) + "/record/1"

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

func NewVizService(port int, game *MockGame, recordFile string) *vizserver.VizService {

	recordStore := recording.NewSingleFileRecordStore(recordFile)

	// TODO(jerome|sven): refac webclient path / serving
	webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"

	vizgames := make([]*types.VizGame, 1)
	vizgames[0] = types.NewVizGame(game)

	mappack, errMappack := mappack.UnzipAndGetHandles(mapcmd.GetMapLocation(mapName))
	if errMappack != nil {
		utils.FailWith(errMappack)
	}

	vizservice := vizserver.NewVizService(
		"0.0.0.0:"+strconv.Itoa(vizport),
		webclientpath,
		mapName,
		func() ([]*types.VizGame, error) { return vizgames, nil },
		recorder,
		mappack,
	)

	return vizservice
}
