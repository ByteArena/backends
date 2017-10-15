package arenamaster

import (
	"fmt"
	"sync"
	"time"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

const (
	TIME_AFTER_UNHEALTHY = 30 * time.Second
)

type LastSeenNodes map[string]time.Time
type MemorizedHealtchecks map[string]bool

type ArenaHealthCheck struct {
	gameHealthcheckRes Res
	ticker             *time.Ticker
	mutex              sync.Mutex
	mqclient           *mq.Client

	lastSeen LastSeenNodes
	cache    MemorizedHealtchecks
}

func NewArenaHealthcheck(gameHealthcheckRes Res, mqclient *mq.Client) *ArenaHealthCheck {
	ticker := time.NewTicker(time.Duration(5) * time.Second)

	instance := &ArenaHealthCheck{
		ticker:             ticker,
		gameHealthcheckRes: gameHealthcheckRes,
		mqclient:           mqclient,

		cache:    make(MemorizedHealtchecks),
		lastSeen: make(LastSeenNodes),
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

		// Check last seen nodes
		now := time.Now()
		for k, date := range s.GetLastSeen() {
			if now.Sub(date) >= TIME_AFTER_UNHEALTHY {
				s.mutex.Lock()
				s.cache[k] = false
				s.mutex.Unlock()
			}
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

			s.lastSeen[mac] = time.Now()

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

func (s ArenaHealthCheck) GetLastSeen() LastSeenNodes {
	return s.lastSeen
}
