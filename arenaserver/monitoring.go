package arenaserver

import (
	"strconv"
	"time"

	"github.com/bytearena/bytearena/common/utils"
)

func (s *Server) monitoring(stopChannel chan bool) {
	monitorfreq := time.Second
	debugNbUpdates := s.currentturn
	for {
		select {
		case <-stopChannel:
			{
				break
			}
		case <-time.After(monitorfreq):
			{
				utils.Debug("monitoring",
					"-- MONITORING -- "+
						strconv.Itoa(int(s.currentturn-debugNbUpdates))+" ticks per "+monitorfreq.String(),
				)

				debugNbUpdates = s.currentturn
			}
		}
	}
}
