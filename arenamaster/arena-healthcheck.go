package arenamaster

import (
	"fmt"
	"sync"
	"time"

	"github.com/xtuc/schaloop"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

const (
	TIME_AFTER_UNHEALTHY = 30 * time.Second
	HEALTHCHECK_FREQ     = 5 * time.Second
)

func timerToGeneric(old *time.Ticker) chan interface{} {
	new := make(chan interface{})
	go func() {
		for {
			<-old.C
			new <- true
		}
	}()

	return new
}

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

	instance := &ArenaHealthCheck{
		gameHealthcheckRes: gameHealthcheckRes,
		mqclient:           mqclient,

		cache:    make(MemorizedHealtchecks),
		lastSeen: make(LastSeenNodes),
	}

	return instance
}

func (s *ArenaHealthCheck) StartChecks(eventloop *schaloop.EventLoop) {
	s.ticker = time.NewTicker(HEALTHCHECK_FREQ)

	s.startConsumer(eventloop)
	s.startTicker(eventloop)
}

func (s *ArenaHealthCheck) startTicker(eventloop *schaloop.EventLoop) {
	eventloop.QueueWorkFromChannel("healtcheck-ticker", timerToGeneric(s.ticker), func(data interface{}) {
		err := s.mqclient.Publish("game", "healthcheck", types.MQPayload{})

		if err != nil {
			utils.RecoverableError("healtcheck", "error: "+err.Error())
		} else {

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
	})
}

func (s *ArenaHealthCheck) startConsumer(eventloop *schaloop.EventLoop) {

	eventloop.QueueWorkFromChannel("healtcheck-consumer", resToGeneric(s.gameHealthcheckRes), func(data interface{}) {
		msg := data.(types.MQMessage)

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
	})
}

// We want to copy here
func (s ArenaHealthCheck) GetCache() MemorizedHealtchecks {
	return s.cache
}

func (s ArenaHealthCheck) GetLastSeen() LastSeenNodes {
	return s.lastSeen
}
