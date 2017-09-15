package arenamaster

import (
	"encoding/json"
	"os"
	"time"

	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"

	"github.com/influxdata/influxdb/client/v2"
)

type ListeningChanStruct chan struct{}
type Server struct {
	listeningChan ListeningChanStruct
	brokerclient  *mq.Client
	graphqlclient *graphql.Client
	state         *State
}

func NewServer(mq *mq.Client, gql *graphql.Client) *Server {
	s := &Server{
		brokerclient:  mq,
		graphqlclient: gql,
		state:         NewState(),
	}

	if os.Getenv("INFLUXDB_ADDR") != "" {
		utils.Debug("arenamaster", "State reporting activated")
		s.startStateReporting(os.Getenv("INFLUXDB_ADDR"), os.Getenv("INFLUXDB_DB"))
	}

	return s
}

func (server *Server) startStateReporting(addr, db string) {
	influxdbClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: addr,
	})

	if err != nil {
		panic(err)
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: db,
	})

	if err != nil {
		panic(err)
	}

	go func() {
		for {
			<-time.NewTicker(5 * time.Second).C

			// Create a point and add to batch
			tags := map[string]string{"arenamaster": "arenamaster"}
			fields := map[string]interface{}{
				"arenamaster.state.idle":    len(server.state.idleArenas),
				"arenamaster.state.running": len(server.state.runningArenas),
				"arenamaster.state.pending": len(server.state.pendingArenas),
			}

			pt, err := client.NewPoint("arenamaster", tags, fields, time.Now())

			if err != nil {
				panic(err.Error())
			} else {
				utils.Debug("state-reporting", "New point: "+pt.String())
			}

			bp.AddPoint(pt)
			influxdbClient.Write(bp)
		}
	}()
}

func unmarshalMQMessage(msg mq.BrokerMessage) (error, *types.MQMessage) {
	var message types.MQMessage
	err := json.Unmarshal(msg.Data, &message)
	if err != nil {
		return err, nil
	} else {
		return nil, &message
	}
}

func (server *Server) Start() ListeningChanStruct {

	server.brokerclient.Subscribe("game", "launch", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
			return
		} else {
			onGameLaunch(server.state, message.Payload, server.brokerclient, server.graphqlclient)
		}
	})

	server.brokerclient.Subscribe("game", "launched", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
			return
		} else {
			onGameLaunched(server.state, message.Payload, server.brokerclient, server.graphqlclient)
		}
	})

	server.brokerclient.Subscribe("game", "handshake", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
			return
		} else {
			onGameHandshake(server.state, message.Payload)
		}
	})

	server.brokerclient.Subscribe("game", "stopped", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
			return
		} else {
			onGameStop(server.state, message.Payload, server.graphqlclient)
		}
	})

	server.listeningChan = make(ListeningChanStruct)

	utils.Debug("arenamaster", "Listening")

	return server.listeningChan
}

func (server *Server) Stop() {
	close(server.listeningChan)
}
