package arenatrainer

import (
	"encoding/json"
	"time"

	"github.com/bytearena/bytearena/common/messagebroker"
)

type MemoryMessageClient struct {
	subscriptions *messagebroker.SubscriptionMap
}

func NewMemoryMessageClient() (*MemoryMessageClient, error) {
	c := &MemoryMessageClient{
		subscriptions: messagebroker.NewSubscriptionMap(),
	}

	return c, nil
}

func (client *MemoryMessageClient) Subscribe(channel string, topic string, onmessage messagebroker.SubscriptionCallback) error {
	client.subscriptions.Set(channel+":"+topic, onmessage)
	return nil
}

func (client *MemoryMessageClient) Publish(channel string, topic string, payload interface{}) error {

	subscription := client.subscriptions.Get(channel + ":" + topic)
	if subscription != nil {
		res, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		go subscription(messagebroker.BrokerMessage{
			Timestamp: time.Now().Format(time.RFC3339),
			Topic:     topic,
			Channel:   channel,
			Data:      res,
		})
	}

	return nil
}
