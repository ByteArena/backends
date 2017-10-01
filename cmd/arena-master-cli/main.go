package main

import (
	"flag"
	"strings"

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

	shell.Println("Sample Interactive Shell")

	shell.AddCmd(&ishell.Cmd{
		Name: "vm/start",
		Help: "Start VM",
		Func: session.handleStartVMCommand,
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "arena/game/start",
		Help: "Start game on a given arena",
		Func: session.handleStartGameCommand,
	})

	shell.Run()
}

func (s Session) handleStartVMCommand(c *ishell.Context) {
	c.Println("Hello", strings.Join(c.Args, " "))
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
