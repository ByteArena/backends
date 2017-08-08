package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	notify "github.com/bitly/go-notify"

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/vizserver"
)

func main() {
	filename := flag.String("file", "", "Filename")
	port := flag.Int("viz-port", 8080, "Specifiy the port of the visualization server")
	debug := flag.Bool("debug", false, "Enable debug mode")

	flag.Parse()

	utils.Assert(*filename != "", "file must be set")

	arenainstance := NewMockArenaInstance(10)

	vizserver := NewVizService(*port, arenainstance)

	vizserver.Start()
	go replay.Read(*filename, *debug, arenainstance.GetId(), sendMessageToViz)

	<-common.SignalHandler()
	utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")
	vizserver.Stop()
}

func sendMessageToViz(msg string, debug bool, UUID string) {
	if debug {
		log.Println("read buffer of length: ", len(msg))
	}

	notify.PostTimeout("viz:message"+UUID, msg, time.Millisecond)
	<-time.NewTimer(1 * time.Second).C
}

func NewVizService(port int, arenainstance *MockArenaInstance) *vizserver.VizService {

	recorder := recording.MakeEmptyRecorder()

	// TODO: refac webclient path / serving
	webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"
	vizservice := vizserver.NewVizService("0.0.0.0:"+strconv.Itoa(port), webclientpath, func() ([]arenaserver.ArenaInstance, error) {
		res := make([]arenaserver.ArenaInstance, 1)
		res[0] = arenainstance
		return res, nil
	}, recorder)

	return vizservice
}