package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/abiosoft/ishell"

	bamq "github.com/bytearena/core/common/mq"
	"github.com/bytearena/backends/common/mq"

	"github.com/bytearena/core/common/types"
	"github.com/bytearena/core/common/utils"
)

type Session struct {
	mqClient *mq.Client
}

func main() {
	mqHost := flag.String("mq", "", "MQ addr")

	flag.Parse()

	utils.Assert(*mqHost != "", "--mq must be set")

	shell := ishell.New()

	mqClient, err := mq.NewClient(*mqHost)
	utils.Check(err, "ERROR: could not connect to messagebroker at "+string(*mqHost))

	session := Session{
		mqClient: mqClient,
	}

	session.mqClient.Subscribe("debug", "getvmstatus-res", func(msg bamq.BrokerMessage) {
		var dat map[string]interface{}

		if err := json.Unmarshal(msg.Data, &dat); err != nil {
			panic(err)
		}

		b, err := json.MarshalIndent(dat, "", "  ")
		utils.Check(err, "Could not prettify JSON")

		fmt.Println(string(b))
	})

	shell.Println("arena-master cli")

	shell.AddCmd(&ishell.Cmd{
		Name: "debug/GetVmStatus",
		Help: "Get VMs status",
		Func: session.handleDebugGetVmStatus,
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "arena/add",
		Help: "Add arena VM",
		Func: session.handleArenaAddCommand,
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "arena/halt",
		Help: "Halt arena VM",
		Func: session.handleArenaHaltCommand,
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "game/start",
		Help: "Start game",
		Func: session.handleStartGameCommand,
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "game/stop",
		Help: "Fake stop game",
		Func: session.handleStopGameCommand,
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "arena/game/start",
		Help: "Start game on a given arena",
		Func: session.handleStartGameOnArenaCommand,
	})

	shell.Run()
}

func (s Session) handleArenaAddCommand(c *ishell.Context) {
	err := s.mqClient.Publish("arena", "add", types.MQPayload{})

	if err != nil {
		c.Println("MQ error: " + err.Error())
	} else {
		c.Println("OK")
	}
}

func (s Session) handleArenaHaltCommand(c *ishell.Context) {
	c.Print("VM ID: ")
	vmId := c.ReadLine()

	err := s.mqClient.Publish("arena", "halt", types.NewMQMessage(
		"arena-master",
		"halt",
	).SetPayload(types.MQPayload{
		"id": vmId,
	}))

	if err != nil {
		c.Println("MQ error: " + err.Error())
	} else {
		c.Println("OK")
	}
}

func (s Session) handleStartGameOnArenaCommand(c *ishell.Context) {
	c.Print("Game ID: ")
	gameId := c.ReadLine()

	c.Print("Arena ID: ")
	arenaId := c.ReadLine()

	err := s.mqClient.Publish("game", arenaId+".launch", types.MQPayload{
		"id": gameId,
	})

	if err != nil {
		c.Println("MQ error: " + err.Error())
	} else {
		c.Println("OK")
	}
}

func (s Session) handleStartGameCommand(c *ishell.Context) {
	c.Print("Game ID: ")
	gameId := c.ReadLine()

	err := s.mqClient.Publish("game", "launch", types.NewMQMessage(
		"arena-master",
		"launch",
	).SetPayload(types.MQPayload{
		"id": gameId,
	}))

	if err != nil {
		c.Println("MQ error: " + err.Error())
	} else {
		c.Println("OK")
	}
}

func (s Session) handleStopGameCommand(c *ishell.Context) {
	c.Print("Game ID: ")
	gameId := c.ReadLine()

	c.Print("Arena MAC: ")
	arenaserveruuid := c.ReadLine()

	err := s.mqClient.Publish("game", "stopped", types.NewMQMessage(
		"arena-master",
		"launch",
	).SetPayload(types.MQPayload{
		"id":              gameId,
		"arenaserveruuid": arenaserveruuid,
	}))

	if err != nil {
		c.Println("MQ error: " + err.Error())
	} else {
		c.Println("OK")
	}
}

func (s Session) handleDebugGetVmStatus(c *ishell.Context) {
	err := s.mqClient.Publish("debug", "getvmstatus", types.NewMQMessage(
		"arena-master",
		"debug",
	))

	if err != nil {
		c.Println("MQ error: " + err.Error())
	} else {
		c.Println("OK")
	}
}
