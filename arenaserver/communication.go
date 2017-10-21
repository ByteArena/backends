package arenaserver

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/utils"
)

func (s *Server) listen() chan interface{} {
	serveraddress := "0.0.0.0:" + strconv.Itoa(s.port)
	s.commserver = comm.NewCommServer(serveraddress)

	utils.Debug("arena", "Server listening on port "+strconv.Itoa(s.port))

	err := s.commserver.Listen(s)
	utils.Check(err, "Failed to listen on "+serveraddress)

	block := make(chan interface{})
	notify.Start("app:stopticking", block)

	return block
}

/* <implementing types.AgentCommunicatorInterface> */
func (s *Server) NetSend(message []byte, conn net.Conn) error {
	return s.commserver.Send(message, conn)
}

func (s *Server) PushMutationBatch(batch arenaservertypes.AgentMutationBatch) {
	s.mutationsmutex.Lock()
	s.pendingmutations = append(s.pendingmutations, batch)
	s.mutationsmutex.Unlock()
}

/* </implementing types.AgentCommunicatorInterface> */

/* <implementing types.CommunicatorDispatcherInterface> */
func (s *Server) ImplementsCommDispatcherInterface() {}
func (s *Server) DispatchAgentMessage(msg arenaservertypes.AgentMessage) error {

	agentproxy, err := s.getAgentProxy(msg.GetAgentId().String())
	if err != nil {
		return errors.New("DispatchAgentMessage: agentid does not match any known agent in received agent message !;" + msg.GetAgentId().String())
	}

	// proto := msg.GetEmitterConn().LocalAddr().Network()
	// ip := strings.Split(msg.GetEmitterConn().RemoteAddr().String(), ":")[0]
	// if proto != "tcp" || ip != "TODO(jerome):take from agent container struct"
	// Problem here: cannot check ip against the one we get from Docker by inspecting the container
	// as the two addresses do not match

	switch msg.GetType() {
	case arenaservertypes.AgentMessageType.Handshake:
		{
			if _, found := s.agentproxieshandshakes[msg.GetAgentId()]; found {
				return errors.New("ERROR: Received duplicate handshake from agent " + agentproxy.String())
			}

			s.agentproxieshandshakes[msg.GetAgentId()] = struct{}{}

			var handshake arenaservertypes.AgentMessagePayloadHandshake
			err = json.Unmarshal(msg.GetPayload(), &handshake)
			if err != nil {
				return errors.New("DispatchAgentMessage: Failed to unmarshal JSON agent handshake payload for agent " + msg.GetAgentId().String() + "; " + string(msg.GetPayload()))
			}

			ag, ok := agentproxy.(agent.AgentProxyNetworkInterface)
			if !ok {
				return errors.New("DispatchAgentMessage: Failed to cast agent to NetAgent during handshake for " + ag.String())
			}

			ag = ag.SetConn(msg.GetEmitterConn())
			s.setAgentProxy(ag)

			utils.Debug("arena", "Received handshake from agent "+ag.String()+"; agent said \""+handshake.GetGreetings()+"\"")

			s.nbhandshaked++

			if s.nbhandshaked == s.getNbExpectedagents() {
				s.onAgentsReady()
			}

			// TODO(sven|jerome): handle some timeout here if all agents fail to handshake

			break
		}
	case arenaservertypes.AgentMessageType.Mutation:
		{
			var mutations struct {
				Mutations []arenaservertypes.AgentMessagePayloadMutation
			}

			err = json.Unmarshal(msg.GetPayload(), &mutations)
			if err != nil {
				return errors.New("DispatchAgentMessage: Failed to unmarshal JSON agent mutation payload for agent " + agentproxy.String() + "; " + string(msg.GetPayload()))
			}

			mutationbatch := arenaservertypes.AgentMutationBatch{
				AgentProxyUUID: agentproxy.GetProxyUUID(),
				AgentEntityId:  agentproxy.GetEntityId(),
				Mutations:      mutations.Mutations,
			}

			s.PushMutationBatch(mutationbatch)

			break
		}
	default:
		{
			return errors.New("DispatchAgentMessage: Unknown message type" + msg.GetType().String())
		}
	}

	return nil
}

/* </implementing types.CommunicatorDispatcherInterface> */
