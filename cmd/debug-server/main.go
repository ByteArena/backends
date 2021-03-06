package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/bytearena/core/arenaserver/container"
	arenaservertypes "github.com/bytearena/core/common/types"
	commontypes "github.com/bytearena/core/common/types"
	"github.com/bytearena/core/common/utils"
)

func main() {

	os.Setenv("CONTAINER_UNIX_USER", "nobody")
	os.Setenv("ENV", "prod")

	dockerimage := "agent/netgusto/debug-agent"
	host := "192.168.0.10"
	port := "9999"

	///////////////////////////////////////////////////////////////////////////
	// Make orchestrator
	///////////////////////////////////////////////////////////////////////////

	// registryAddr := "https://registry.net.bytearena.com"
	// arenaAddr := "192.168.0.10"
	// orch := container.MakeRemoteContainerOrchestrator(arenaAddr, registryAddr)

	orch := container.MakeLocalContainerOrchestrator("")

	///////////////////////////////////////////////////////////////////////////
	// Spawn agent
	///////////////////////////////////////////////////////////////////////////

	agentid := uuid.NewV4()
	intport, _ := strconv.Atoi(port)

	container, err := orch.CreateAgentContainer(
		agentid,
		host,
		intport,
		dockerimage,
	)
	if err != nil {
		utils.Debug("debug-server", "Failed to create docker container for "+agentid.String()+": "+err.Error())
		os.Exit(1)
	}

	err = orch.StartAgentContainer(container, func(commontypes.TearDownCallback) {})

	if err != nil {
		utils.Debug("debug-server", "Failed to start docker container for "+agentid.String()+": "+err.Error())
		os.Exit(1)
	}

	///////////////////////////////////////////////////////////////////////////
	// TCP stack
	///////////////////////////////////////////////////////////////////////////

	go func() {
		listenAddress := host + ":" + port

		ln, err := net.Listen("tcp4", listenAddress)
		if err != nil {
			utils.Debug("debug-server", fmt.Sprintf("Comm server could not listen on %s; %s", listenAddress, err.Error()))
			os.Exit(1)
		}

		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				utils.Debug("debug-server", err.Error())
				os.Exit(1)
			}

			go func() {
				defer conn.Close()
				for {

					reader := bufio.NewReader(conn)
					buf, err := reader.ReadBytes('\n')
					if err != nil {
						// Avoid crashes when agent crashes Issue #108
						utils.Debug("commserver", "Connexion closed unexpectedly; "+err.Error())
						return
					}

					// Unmarshal message (unwrapping in an AgentMessage structure)
					var msg arenaservertypes.AgentMessage
					err = json.Unmarshal(buf, &msg)
					if err != nil {
						utils.Debug("commserver", "Failed to unmarshal incoming JSON in CommServer::Listen(); "+string(buf)+";"+err.Error())
						return
					}

					msg.EmitterConn = conn

					go func() {
						utils.Debug("arena-trainer", "Dispatching agent message"+string(msg.GetPayload()))
					}()
				}
			}()
		}
	}()

	utils.Debug("debug-server", "RUNNING !")
	time.Sleep(time.Second * 60)
	utils.Debug("debug-server", "EXITING !")
}
