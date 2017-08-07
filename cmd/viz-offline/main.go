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

	vizserver := NewVizService(*port)

	vizserver.Start()
	go read(*filename, *debug)

	<-common.SignalHandler()
	utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")
	vizserver.Stop()
}

func read(filename string, debug bool) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0755)

	utils.CheckWithFunc(err, func() string {
		return "File open failed: " + err.Error()
	})

	reader := bufio.NewReader(file)

	for {
		line, isPrefix, readErr := reader.ReadLine()

		if len(line) == 0 {
			continue
		}

		if readErr == io.EOF {
			return
		}

		if !isPrefix {
			sendMessageToViz(string(line), debug)
		} else {
			buf := append([]byte(nil), line...)
			for isPrefix && err == nil {
				line, isPrefix, err = reader.ReadLine()
				buf = append(buf, line...)
			}

			sendMessageToViz(string(buf), debug)
		}
	}
}

func sendMessageToViz(msg string, debug bool) {
	if debug {
		log.Println("read buffer of length: ", len(msg))
	}

	notify.PostTimeout("viz:message", msg, time.Millisecond)
	<-time.NewTimer(1 * time.Second).C
}

func NewVizService(port int) *vizserver.VizService {

	recorder := recording.MakeEmptyRecorder()
	arenainstance := NewMockArenaInstance(10)

	// TODO: refac webclient path / serving
	webclientpath := utils.GetExecutableDir() + "/../viz-server/webclient/"
	vizservice := vizserver.NewVizService("0.0.0.0:"+strconv.Itoa(port), webclientpath, func() ([]arenaserver.ArenaInstance, error) {
		res := make([]arenaserver.ArenaInstance, 1)
		res[0] = arenainstance
		return res, nil
	}, recorder)

	return vizservice
}