package mq

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/bytearena/core/common/mq"
	"github.com/bytearena/core/common/utils"

	"github.com/go-redis/redis"
)

type brokerAction struct {
	Action  string      `json:"action"`
	Channel string      `json:"channel"`
	Topic   string      `json:"topic"`
	Data    interface{} `json:"data"`
}

type Client struct {
	conn          *redis.Client
	subscriptions map[string]*redis.PubSub
	mu            sync.Mutex
	host          string
	isClosed      bool
}

func NewClient(host string) (*Client, error) {
	c := &Client{
		conn:          nil,
		subscriptions: make(map[string]*redis.PubSub, 0),
		host:          host,
		isClosed:      false,
	}

	hasConnected := c.connect()
	utils.Assert(hasConnected, "Error: cannot connect to messagebroker host "+host)

	return c, nil
}

func channelAndTopicToString(channel, topic string) string {
	return channel + "." + topic
}

func (client *Client) connect() bool {
	conn := redis.NewClient(&redis.Options{
		Addr:     client.host + ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if conn == nil {
		return false
	}

	client.conn = conn

	return true
}

func (client *Client) Stop() {
	client.isClosed = true

	// Stop all current Redis PubSub subscriptions
	for _, pubsub := range client.subscriptions {
		pubsub.Close()
	}

	err := client.conn.Close()
	utils.Check(err, "Unable to Redis client connection")
}

/* <mq.MessageBrokerClientInterface> */
func (client *Client) Subscribe(channel string, topic string, onmessage mq.SubscriptionCallback) error {
	client.mu.Lock()

	channelName := channelAndTopicToString(channel, topic)

	pubsub := client.conn.Subscribe(channelName)

	client.mu.Unlock()

	if pubsub == nil {
		return errors.New("Could not subscribe to channel " + channelName)
	}

	utils.Debug("mq", "Subscribed to bus "+channelName)

	client.subscriptions[channelName] = pubsub

	/*
		Handle message loop
	*/
	go func() {
		for {
			if client.isClosed == true {
				break
			}

			msg, err := pubsub.ReceiveMessage()

			if err != nil {
				utils.RecoverableError("mqclient", "Could not receive message: "+err.Error())
				continue
			}

			var mqMessage mq.BrokerMessage

			err = json.Unmarshal([]byte(msg.Payload), &mqMessage)

			if err != nil {
				utils.RecoverableError("mqclient", "Received invalid message; "+err.Error()+";"+msg.Payload)
				continue
			}

			onmessage(mqMessage)
		}
	}()

	return nil
}

func (client *Client) Publish(channel, topic string, payload interface{}) error {
	channelName := channelAndTopicToString(channel, topic)

	client.mu.Lock()

	brokerAction := brokerAction{
		Action:  "pub",
		Channel: channel,
		Topic:   topic,
		Data:    payload,
	}

	jsonPayload, err := json.Marshal(brokerAction)

	if err != nil {
		client.mu.Unlock()
		return err
	}

	res := client.conn.Publish(channelName, string(jsonPayload))
	client.mu.Unlock()

	if res.Err() != nil {
		return res.Err()
	}

	return nil
}

func (client *Client) Ping() error {
	_, err := client.conn.Ping().Result()

	return err
}

/* </mq.MessageBrokerClientInterface> */
