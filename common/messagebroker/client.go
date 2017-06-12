package messagebroker

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bytearena/bytearena/utils"
	"github.com/gorilla/websocket"
)

type brokerAction struct {
	Action  string      `json:"action"`
	Channel string      `json:"channel"`
	Topic   string      `json:"topic"`
	Data    interface{} `json:"data"`
}

type BrokerMessage struct {
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
	Topic     string          `json:"topic"`
	Channel   string          `json:"channel"`
}

type Client struct {
	conn          *websocket.Conn
	subscriptions *SubscriptionMap
}

func NewClient(host string) (*Client, error) {

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial("ws://"+host, http.Header{})
	if err != nil {
		return nil, errors.New("Error: cannot connect to messagebroker host " + host)
	}

	c := &Client{
		conn:          conn,
		subscriptions: NewSubscriptionMap(),
	}

	go c.waitAndListen()
	return c, nil
}

func (client *Client) waitAndListen() {
	for {
		_, rawData, err := client.conn.ReadMessage()
		utils.Check(err, "Received invalid message")

		var message BrokerMessage

		err = json.Unmarshal(rawData, &message)
		utils.Check(err, "Received invalid message")

		subscription := client.subscriptions.Get(message.Channel + ":" + message.Topic)
		utils.Assert(
			subscription != nil,
			"unexpected (unsubscribed) message type "+message.Channel+":"+message.Topic,
		)

		subscription(message)
	}
}

/* <messagebroker.MessageBrokerClientInterface> */
func (client *Client) Subscribe(channel string, topic string, onmessage SubscriptionCallback) error {
	err := client.conn.WriteJSON(brokerAction{
		Action:  "sub",
		Channel: channel,
		Topic:   topic,
	})

	if err != nil {
		return errors.New("Error: cannot subscribe to message broker (" + channel + ", " + topic + ")")
	}

	client.subscriptions.Set(channel+":"+topic, onmessage)

	return nil
}

func (client *Client) Publish(channel string, topic string, payload interface{}) error {
	err := client.conn.WriteJSON(brokerAction{
		Action:  "pub",
		Channel: channel,
		Topic:   topic,
		Data:    payload,
	})

	if err != nil {
		return errors.New("Error: cannot publish to message broker (" + channel + ", " + topic + ")")
	}

	return nil
}

/* </messagebroker.MessageBrokerClientInterface> */
