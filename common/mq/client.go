package mq

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/bytearena/bytearena/common/utils"
	"github.com/cenkalti/backoff"
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
	mu            sync.Mutex
	host          string
}

func NewClient(host string) (*Client, error) {
	c := &Client{
		conn:          nil,
		subscriptions: NewSubscriptionMap(),
		host:          host,
	}

	hasConnected := c.connect()
	utils.Assert(hasConnected, "Error: cannot connect to messagebroker host "+host)

	go c.waitAndListen()

	return c, nil
}

func (client *Client) connect() bool {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial("wss://"+client.host, http.Header{})
	if err != nil {
		return false
	}

	client.conn = conn

	return true
}

func handleUnexepectedClose(client *Client) {
	utils.Debug("mq-client", "Unexpected close")

	f := func() error {
		utils.Debug("mq-client", "Try to reconnect")
		hasConnected := client.connect()

		if hasConnected {
			utils.Debug("mq-client", "Reconnected")
			for _, subscriptionLane := range client.subscriptions.GetKeys() {
				subscriptionCbk := client.subscriptions.Get(subscriptionLane)
				parts := strings.Split(subscriptionLane, ":")
				utils.Debug("mq-client", "Re-subscribing to "+subscriptionLane)
				client.Subscribe(parts[0], parts[1], subscriptionCbk)
			}
			return nil
		}

		return errors.New("connection failed")
	}

	backoff.Retry(f, backoff.NewExponentialBackOff())
}

func (client *Client) waitAndListen() {
	for {
		_, rawData, err := client.conn.ReadMessage()

		if websocket.IsUnexpectedCloseError(err) {
			handleUnexepectedClose(client)

			continue
		}

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

/* <mq.MessageBrokerClientInterface> */
func (client *Client) Subscribe(channel string, topic string, onmessage SubscriptionCallback) error {
	client.mu.Lock()

	err := client.conn.WriteJSON(brokerAction{
		Action:  "sub",
		Channel: channel,
		Topic:   topic,
	})

	client.mu.Unlock()

	if err != nil {
		return errors.New("Error: cannot subscribe to message broker (" + channel + ", " + topic + ")")
	}

	client.subscriptions.Set(channel+":"+topic, onmessage)

	return nil
}

func (client *Client) Publish(channel string, topic string, payload interface{}) error {
	client.mu.Lock()

	err := client.conn.WriteJSON(brokerAction{
		Action:  "pub",
		Channel: channel,
		Topic:   topic,
		Data:    payload,
	})

	client.mu.Unlock()

	if err != nil {
		return errors.New("Error: cannot publish to message broker (" + channel + ", " + topic + ")")
	}

	return nil
}

func (client *Client) Ping() (err error) {
	var data interface{}
	return client.Publish("ping", "ping", data)
}

/* </mq.MessageBrokerClientInterface> */
