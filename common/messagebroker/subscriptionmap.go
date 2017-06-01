package messagebroker

import (
	commontypes "github.com/bytearena/bytearena/common/types"
)

type subscriptionCallback func(msg BrokerMessage)

type subscriptionMap struct {
	*commontypes.SyncMap
}

func newSubscriptionMap() *subscriptionMap {
	return &subscriptionMap{
		commontypes.NewSyncMap(),
	}
}

func (smap *subscriptionMap) Get(id string) subscriptionCallback {
	if res, ok := (smap.GetGeneric(id)).(subscriptionCallback); ok {
		return res
	}

	return nil
}
