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
		s.startStateReporting(os.Getenv("INFLUXDB_ADDR"), os.Getenv("INFLUXDB_DB"))
	}

	return s
}

func (server *Server) startStateReporting(addr, db string) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: addr,
	})

	if err != nil {
		panic(err)
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  db,
		Precision: "s",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		for {
			<-time.NewTicker(5 * time.Second).C

			// Create a point and add to batch
			tags := map[string]string{"cpu": "cpu-total"}
			fields := map[string]interface{}{
				"idle":    server.state.idleArenas,
				"running": server.state.runningArenas,
				"pending": server.state.pendingArenas,
			}

			utils.Debug("arenamaster", "Reporting state")

			pt, err := client.NewPoint("cpu_usage", tags, fields, time.Now())

			if err != nil {
				panic(err.Error())
			}

			bp.AddPoint(pt)

			c.Write(bp)
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
