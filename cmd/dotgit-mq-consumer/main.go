package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/common/messagebroker"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/dotgit/config"
	"github.com/bytearena/bytearena/dotgit/database"
	"github.com/bytearena/bytearena/dotgit/protocol"
	dotgitutils "github.com/bytearena/bytearena/dotgit/utils"
)

type messageAgentSubmitted struct {
	Id string `json:"id"`
}

func main() {

	cnf := config.GetConfig()

	f, err := os.OpenFile("/var/log/dotgit-mq-consumer.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println("Starting dotgit-mq-consumer daemon")

	var db protocol.Database = database.NewGraphQLDatabase()
	err = db.Connect(cnf.GetDatabaseURI())
	if err != nil {
		fmt.Println("Cannot connect to database")
		log.Println("Cannot connect to database")
		f.Close()
		os.Exit(1)
	}

	brokerclient, err := messagebroker.NewClient(cnf.GetMqHost())
	utils.Check(err, "ERROR: could not connect to messagebroker")

	streamAgentSubmitted := make(chan interface{})
	notify.Start("agent:submitted", streamAgentSubmitted)

	brokerclient.Subscribe("agent", "submitted", func(msg messagebroker.BrokerMessage) {
		log.Println("INFO:agent:submitted Received from MESSAGEBROKER")
		var payload messageAgentSubmitted
		err := json.Unmarshal(msg.Data, &payload)
		if err != nil {
			log.Println(err)
			log.Println("ERROR:agent:submitted Invalid payload " + string(msg.Data))
			return
		}

		notify.PostTimeout("agent:submitted", payload, time.Millisecond)
	})

	go func() {
		for {
			select {
			case payload := <-streamAgentSubmitted:
				{
					if agentSubmitted, ok := payload.(messageAgentSubmitted); ok {
						go initRepo(db, brokerclient, agentSubmitted.Id)
					}
				}
			}
		}
	}()

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)

	<-hassigtermed
}

func initRepo(db protocol.Database, mq messagebroker.ClientInterface, agentid string) {
	// fetch de l'agent sur graphql
	agent, err := db.FindRepositoryById(agentid)
	if err != nil {
		errmsg := "ERROR:agent:submitted Could not fetch agent by id '" + agentid + "'"
		log.Println(errmsg)
		log.Println(err)
		mq.Publish(
			"agent", "repo-init-fail", utils.NewMQError(
				"dotgit-mq-consumer",
				errmsg,
			).SetPayload(utils.MQPayload{
				"agentid": agentid,
			}),
		)
		return
	}

	// création du repo via git init --bare
	err = dotgitutils.InitBareGitRepository(agent)
	if err != nil {
		errmsg := "ERROR:agent:submitted Could not fetch agent by id '" + agentid + "'"
		log.Println(errmsg)
		mq.Publish(
			"agent", "repo-init-fail", utils.NewMQError(
				"dotgit-mq-consumer",
				errmsg,
			).SetPayload(utils.MQPayload{
				"agentid": agentid,
			}),
		)
		return
	}

	// appel de mq
	mq.Publish(
		"agent", "repo-init-success", utils.NewMQMessage(
			"dotgit-mq-consumer",
			"Git Repo "+agent.Owner.Username+"/"+agent.RepoName+" has been successfuly initialized.",
		).SetPayload(utils.MQPayload{
			"agentid": agentid,
		}),
	)
}
