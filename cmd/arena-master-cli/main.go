package main

import (
	"flag"

	"github.com/abiosoft/ishell"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
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

	shell.Println("arena-master cli")

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
		Name: "arena/game/start",
		Help: "Start game on a given arena",
		Func: session.handleStartGameCommand,
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

	err := s.mqClient.Publish("arena", "halt", types.MQPayload{
		"id": vmId,
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
