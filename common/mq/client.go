package mq

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
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

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial("ws://"+host, http.Header{})
	if err != nil {
		return nil, errors.New("Error: cannot connect to messagebroker host " + host)
	}

	c := &Client{
		conn:          conn,
		subscriptions: NewSubscriptionMap(),
		host:          host,
	}

	go c.waitAndListen()
	return c, nil
}

func (client *Client) tryReconnect() bool {
	log.Println("tryReconnect")

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial("ws://"+client.host, http.Header{})
	if err != nil {
		return false
	}

	client.conn = conn

	return true
}

func (client *Client) waitAndListen() {
	for {
		_, rawData, err := client.conn.ReadMessage()

		if websocket.IsUnexpectedCloseError(err) {
			log.Println("Unexpected close")

			f := func() error {
				res := client.tryReconnect()

				if res == true {
					return nil
				} else {
					return errors.New("error")
				}
			}

			backoff.Retry(f, backoff.NewExponentialBackOff())

			log.Println("Reconnected")
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

func (client *Client) write(msg brokerAction) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	return client.conn.WriteJSON(msg)
}

/* <mq.MessageBrokerClientInterface> */
func (client *Client) Subscribe(channel string, topic string, onmessage SubscriptionCallback) error {
	err := client.write(brokerAction{
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
	err := client.write(brokerAction{
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

func (client *Client) Ping() (err error) {
	var data interface{}
	return client.Publish("ping", "ping", data)
}

/* </mq.MessageBrokerClientInterface> */
