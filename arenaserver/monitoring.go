package arenaserver

import (
	"strconv"
	"time"

	"github.com/bytearena/bytearena/common/utils"
)

func (s *Server) monitoring(stopChannel chan bool) {
	monitorfreq := time.Second
	debugNbMutations := 0
	debugNbUpdates := 0
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
						strconv.Itoa(s.debugNbMutations-debugNbMutations)+" mutations per "+monitorfreq.String()+";"+
						strconv.Itoa(s.debugNbUpdates-debugNbUpdates)+" updates per "+monitorfreq.String(),
				)

				debugNbMutations = s.debugNbMutations
				debugNbUpdates = s.debugNbUpdates

			}
		}
	}
}
