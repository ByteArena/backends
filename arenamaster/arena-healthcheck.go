package arenamaster

import (
	"fmt"
	"sync"
	"time"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

type MemorizedHealtchecks map[string]bool

type ArenaHealthCheck struct {
	gameHealthcheckRes Res
	cache              MemorizedHealtchecks
	ticker             *time.Ticker
	mutex              sync.Mutex
	mqclient           *mq.Client
}

func NewArenaHealthcheck(gameHealthcheckRes Res, mqclient *mq.Client) *ArenaHealthCheck {
	ticker := time.NewTicker(time.Duration(5) * time.Second)

	instance := &ArenaHealthCheck{
		cache:              make(MemorizedHealtchecks),
		ticker:             ticker,
		gameHealthcheckRes: gameHealthcheckRes,
		mqclient:           mqclient,
	}

	go instance.startTicker()
	go instance.startConsumer()

	return instance
}

func (s *ArenaHealthCheck) startTicker() {
	for {
		<-s.ticker.C

		err := s.mqclient.Publish("game", "healthcheck", types.MQPayload{})

		if err != nil {
			utils.RecoverableError("healtcheck", "error: "+err.Error())
			continue
		}
	}
}

func (s *ArenaHealthCheck) startConsumer() {

	for {
		select {
		case msg := <-s.gameHealthcheckRes:
			mac, _ := (*msg.Payload)["id"].(string)
			res, _ := (*msg.Payload)["health"].(string)

			utils.Debug("healthcheck", fmt.Sprintf("Arena %s reported health %s", mac, res))

			s.mutex.Lock()

			if res == "NOK" {
				s.cache[mac] = false
			} else {
				s.cache[mac] = true
			}

			s.mutex.Unlock()
		}
	}
}

// We want to copy here
func (s ArenaHealthCheck) GetCache() MemorizedHealtchecks {
	return s.cache
}
