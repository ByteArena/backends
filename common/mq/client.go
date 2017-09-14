package mq

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/bytearena/bytearena/common/utils"

	"github.com/go-redis/redis"
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
	conn          *redis.Client
	subscriptions map[string]*redis.PubSub
	mu            sync.Mutex
	host          string
}

func NewClient(host string) (*Client, error) {
	c := &Client{
		conn:          nil,
		subscriptions: make(map[string]*redis.PubSub, 0),
		host:          host,
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

// func handleUnexepectedClose(client *Client) {
// 	utils.Debug("mq-client", "Unexpected close")

// 	f := func() error {
// 		utils.Debug("mq-client", "Try to reconnect")
// 		hasConnected := client.connect()

// 		if hasConnected {
// 			utils.Debug("mq-client", "Reconnected")
// 			for _, subscriptionLane := range client.subscriptions.GetKeys() {
// 				subscriptionCbk := client.subscriptions.Get(subscriptionLane)
// 				parts := strings.Split(subscriptionLane, ":")
// 				utils.Debug("mq-client", "Re-subscribing to "+subscriptionLane)
// 				client.Subscribe(parts[0], parts[1], subscriptionCbk)
// 			}
// 			return nil
// 		}

// 		return errors.New("connection failed")
// 	}

// 	backoff.Retry(f, backoff.NewExponentialBackOff())
// }

// func (client *Client) waitAndListen() {

// 	for {
// 		_, rawData, err := client.conn.ReadMessage()

// 		if websocket.IsUnexpectedCloseError(err) {
// 			handleUnexepectedClose(client)
// 			continue
// 		}

// 		if err != nil {
// 			utils.Debug("mqclient", "Received invalid message; "+err.Error())
// 			continue
// 		}

// 		var message BrokerMessage

// 		err = json.Unmarshal(rawData, &message)
// 		if err != nil {
// 			utils.Debug("mqclient", "Received invalid message; "+err.Error()+";"+string(rawData))
// 			continue
// 		}

// 		subscription := client.subscriptions.Get(message.Channel + ":" + message.Topic)
// 		if subscription == nil {
// 			utils.Debug("mqclient", "unexpected (unsubscribed) message type "+message.Channel+":"+message.Topic)
// 			continue
// 		}

// 		subscription(message)
// 	}
// }

/* <mq.MessageBrokerClientInterface> */
func (client *Client) Subscribe(channel string, topic string, onmessage SubscriptionCallback) error {
	client.mu.Lock()

	// err := client.conn.WriteJSON(brokerAction{
	// 	Action:  "sub",
	// 	Channel: channel,
	// 	Topic:   topic,
	// })

	channelName := channelAndTopicToString(channel, topic)

	pubsub := client.conn.Subscribe(channelName)

	client.mu.Unlock()

	if pubsub == nil {
		return errors.New("Could not subscribe to channel " + channelName)
	}

	utils.Debug("mq", "Subscribed to bus "+channelName)

	// if err != nil {
	// 	return errors.New("Error: cannot subscribe to message broker (" + channel + ", " + topic + ")")
	// }

	// client.subscriptions.Set(channel+":"+topic, onmessage)
	client.subscriptions[channelName] = pubsub

	/*
		Handle message loop
	*/
	go func() {
		for {
			fmt.Println("Polling", channelName, "...")

			msg, err := pubsub.ReceiveMessage()
			if err != nil {
				panic(err)
			}

			fmt.Println("message", msg.Channel, msg.Payload)
		}
	}()

	return nil
}

func (client *Client) Publish(channel, topic string, payload interface{}) error {
	channelName := channelAndTopicToString(channel, topic)

	client.mu.Lock()

	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	fmt.Println("publishing", channelName, string(jsonPayload))

	res := client.conn.Publish(channelName, string(jsonPayload))
	client.mu.Unlock()

	if res.Err() != nil {
		return res.Err()
	}

	fmt.Println("publish message", channelName, string(jsonPayload))

	return nil
}

func (client *Client) Ping() error {
	_, err := client.conn.Ping().Result()

	return err
}

/* </mq.MessageBrokerClientInterface> */
