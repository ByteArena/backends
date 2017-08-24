package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/bytearena/bytearena/arenaserver/container"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	uuid "github.com/satori/go.uuid"
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
		panic("Failed to create docker container for " + agentid.String() + ": " + err.Error())
	}

	err = orch.StartAgentContainer(container, func(types.TearDownCallback) {})

	if err != nil {
		panic("Failed to start docker container for " + agentid.String() + ": " + err.Error())
	}

	///////////////////////////////////////////////////////////////////////////
	// TCP stack
	///////////////////////////////////////////////////////////////////////////

	go func() {
		listenAddress := host + ":" + port

		ln, err := net.Listen("tcp4", listenAddress)
		if err != nil {
			panic(fmt.Sprintf("Comm server could not listen on %s; %s", listenAddress, err.Error()))
		}

		utils.Debug("commserver", "::Listen")

		defer ln.Close()
		for {
			utils.Debug("commserver", "::AcceptWaiting")
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}

			utils.Debug("commserver", "::AcceptED")

			go func() {
				defer conn.Close()
				for {
					utils.Debug("commserver", "::Reading...")
					reader := bufio.NewReader(conn)
					buf, err := reader.ReadBytes('\n')
					if err != nil {
						// Avoid crashes when agent crashes Issue #108
						utils.Debug("commserver", "Connexion closed unexpectedly; "+err.Error())
						return
					}

					utils.Debug("commserver", "::RECEIVED bytes"+string(buf))

					// Unmarshal message (unwrapping in an AgentMessage structure)
					var msg protocol.MessageWrapperImp
					err = json.Unmarshal(buf, &msg)
					if err != nil {
						utils.Debug("commserver", "Failed to unmarshal incoming JSON in CommServer::Listen(); "+string(buf)+";"+err.Error())
						return
					}

					msg.EmitterConn = conn

					go func() {
						log.Println("Dispatching agent message", string(msg.GetPayload()))
					}()
				}
			}()
		}
	}()

	log.Println("RUNNING !")
	time.Sleep(time.Second * 60)
	log.Println("EXITING")
}
