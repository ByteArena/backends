package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/ttacon/chalk"
)

func splitEventSlug(eventslug string) (channel string, topic string, err error) {
	res := strings.Split(eventslug, ":")
	if len(res) != 2 || strings.TrimSpace(res[0]) == "" || strings.TrimSpace(res[1]) == "" {
		return "", "", errors.New("Invalid event slug")
	}

	return strings.TrimSpace(res[0]), strings.TrimSpace(res[1]), nil
}

func main() {

	mqHost := flag.String("mqhost", "", "MQ host")
	publish := flag.String("publish", "", "Published event; example agent:repo.pushed")
	publishdata := flag.String("data", "", "Published payload, json; example {\"id\": 5}")

	flag.Parse()

	if *mqHost == "" {
		fmt.Println("Error: mqhost is required.")
		os.Exit(1)
	}

	if *publish == "" {
		fmt.Println("Error: --publish is missing.")
		os.Exit(1)
	}

	brokerclient, err := mq.NewClient(*mqHost)
	utils.Check(err, "Error: could not connect to messagebroker at "+string(*mqHost))

	channel, topic, err := splitEventSlug(*publish)
	utils.Check(err, "Error: Invalid event slug \""+*publish+"\"")

	var payload types.MQPayload
	if *publishdata != "" {
		err = json.Unmarshal([]byte(*publishdata), &payload)
		if err != nil {
			fmt.Println("Error: Invalid json for --data")
			return
		}
	}

	mqmessage := types.
		NewMQMessage("mq-cli", "Synthesizing event from cli").
		SetPayload(payload)

	err = brokerclient.Publish(channel, topic, mqmessage)
	if err != nil {
		panic(err)
	}

	fmt.Print("Message published ")
	fmt.Print(chalk.Yellow)
	fmt.Print(channel+":"+topic, chalk.Reset)
	if payload != nil {
		reencodedpayload, _ := json.Marshal(payload)
		fmt.Print(chalk.Cyan, " ")
		fmt.Print(string(reencodedpayload), chalk.Reset)
	}

	fmt.Println("")

	brokerclient.Stop()
}
