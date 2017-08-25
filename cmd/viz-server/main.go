package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	notify "github.com/bitly/go-notify"

	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/graphql"
	apiqueries "github.com/bytearena/bytearena/common/graphql/queries"
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/vizserver"
	"github.com/bytearena/bytearena/vizserver/types"
)

// Simplified version of the VizMessage struct
type GameIDVizMessage struct {
	GameID          string
	ArenaServerUUID string
}

type GameStoppedMessage struct {
	Payload struct {
		ArenaServerUUID string `json:"arenaserveruuid"`
	} `json:"payload"`
}

type GameListSynchronizer struct {
	gql        graphql.Client
	games      map[string]*types.VizGame
	gamesmutex *sync.RWMutex
	pollfreq   time.Duration
}

func NewGameList(gql graphql.Client, pollfreq time.Duration) *GameListSynchronizer {
	return &GameListSynchronizer{
		gql:        gql,
		games:      make(map[string]*types.VizGame),
		gamesmutex: &sync.RWMutex{},
		pollfreq:   pollfreq,
	}
}

func (glist *GameListSynchronizer) StartSync() {
	pollstop := make(chan interface{})
	notify.Start("poll:stop", pollstop)

	// On initialise la liste immédiatement
	go glist.doFetchFromGQL()

	go func() {

		for {
			select {
			case <-pollstop:
				{
					return
				}
			case <-time.After(glist.pollfreq):
				{
					go glist.doFetchFromGQL()
				}
			}
		}

	}()
}

func (glist *GameListSynchronizer) StopSync() {
	notify.PostTimeout("poll:stop", nil, time.Millisecond*5)
}

func (glist *GameListSynchronizer) doFetchFromGQL() {

	games, err := apiqueries.FetchGames(glist.gql)
	if err != nil {
		utils.Debug("viz-server", "Could not fetch games from GraphQL server")
		return
	}

	glist.gamesmutex.Lock()
	for _, game := range games {
		_, ok := glist.games[game.GetId()]
		if !ok {
			utils.Debug("viz-server", "Serving a new game "+game.GetName()+" with "+strconv.Itoa(len(game.GetContestants()))+" contestants (ID="+game.GetId()+", TPS="+strconv.Itoa(game.GetTps())+")")
			glist.games[game.GetId()] = types.NewVizGame(game)
		}
	}
	glist.gamesmutex.Unlock()
}

func (glist *GameListSynchronizer) GetGameById(gameid string) (game *types.VizGame, ok bool) {
	glist.gamesmutex.RLock()
	game, ok = glist.games[gameid]
	glist.gamesmutex.RUnlock()
	return game, ok
}

func (glist *GameListSynchronizer) GetGames() []*types.VizGame {
	res := make([]*types.VizGame, len(glist.games))

	i := 0
	glist.gamesmutex.RLock()
	for _, game := range glist.games {
		res[i] = game
		i++
	}
	glist.gamesmutex.RUnlock()

	return res
}

func main() {
	env := os.Getenv("ENV")

	// => Serveur HTTP
	//		=> Service des assets statiques de la viz (js, modèles, textures)
	// 		=> Ecoute des messages du messagebroker sur le canal viz
	// 		=> Redistribution des messages via websocket
	// 			=> gestion d'un pool de connexions websocket

	webclientpath := utils.GetExecutableDir() + "/webclient/"

	log.Println("Byte Arena Viz Server v0.1; serving assets from " + webclientpath)

	port := flag.Int("port", 8081, "Port of the viz server")
	mqhost := flag.String("mqhost", "mq:5678", "Message queue host:port")
	apiurl := flag.String("apiurl", "https://graphql.net.bytearena.com", "GQL API URL")
	recordDirectory := flag.String("record-dir", "", "Record files destination")

	flag.Parse()

	// Connect to Message broker
	mqclient, err := mq.NewClient(*mqhost)
	utils.Check(err, "ERROR: could not connect to messagebroker")

	var recorder recording.Recorder = recording.MakeEmptyRecorder()
	if *recordDirectory != "" {
		recorder = recording.MakeMultiArenaRecorder(*recordDirectory)
	}

	// Make GraphQL client
	graphqlclient := graphql.MakeClient(*apiurl)

	// On lance une routine de fetch des games 1x/10 sec
	gamelist := NewGameList(graphqlclient, time.Second*10)
	gamelist.StartSync()

	serverAddr := ":" + strconv.Itoa(*port)
	vizservice := vizserver.NewVizService(serverAddr, webclientpath, func() ([]*types.VizGame, error) {
		return gamelist.GetGames(), nil
	}, recorder)

	mqclient.Subscribe("viz", "message", func(msg mq.BrokerMessage) {
		var vizMessage []GameIDVizMessage
		err := json.Unmarshal([]byte(msg.Data), &vizMessage)

		utils.CheckWithFunc(err, func() string {
			return "Failed to decode vizmessage: " + err.Error()
		})

		gameID := vizMessage[0].GameID
		arenaServerUUID := vizMessage[0].ArenaServerUUID
		game, ok := gamelist.GetGameById(gameID)

		if ok {
			recorder.RecordMetadata(arenaServerUUID, game.GetGame().GetMapContainer())
			recorder.Record(arenaServerUUID, string(msg.Data))
		}

		utils.Debug("viz:message", "received batch of "+strconv.Itoa(len(vizMessage))+" message(s) for arena server "+arenaServerUUID)
		notify.PostTimeout("viz:message:"+gameID, string(msg.Data), time.Millisecond)
	})

	mqclient.Subscribe("game", "stopped", func(msg mq.BrokerMessage) {
		var message GameStoppedMessage
		err := json.Unmarshal(msg.Data, &message)

		utils.CheckWithFunc(err, func() string {
			return "Failed to decode vizmessage: " + err.Error()
		})

		recorder.Close(message.Payload.ArenaServerUUID)
	})

	vizservice.Start()

	var hc *healthcheck.HealthCheckServer
	if env == "prod" {
		hc = NewHealthCheck(mqclient, graphqlclient, "http://"+serverAddr)
		hc.Start()
	}

	<-common.SignalHandler()
	utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")
	vizservice.Stop()

	recorder.Stop()

	if hc != nil {
		hc.Stop()
	}
}
