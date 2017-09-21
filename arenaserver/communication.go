package arenaserver

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	"github.com/bytearena/bytearena/arenaserver/protocol"
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

/* <implementing protocol.AgentCommunicator> */
func (s *Server) NetSend(message []byte, conn net.Conn) error {
	return s.commserver.Send(message, conn)
}

func (s *Server) PushMutationBatch(batch protocol.StateMutationBatch) {
	s.state.PushMutationBatch(batch)
}

/* </implementing protocol.AgentCommunicator> */

/* <implementing protocol.CommunicatorDispatcherInterface> */
func (s *Server) ImplementsCommDispatcherInterface() {}
func (s *Server) DispatchAgentMessage(msg protocol.MessageWrapperInterface) error {

	ag, err := s.getAgent(msg.GetAgentId().String())
	if err != nil {
		return errors.New("DispatchAgentMessage: agentid does not match any known agent in received agent message !;" + msg.GetAgentId().String())
	}

	// proto := msg.GetEmitterConn().LocalAddr().Network()
	// ip := strings.Split(msg.GetEmitterConn().RemoteAddr().String(), ":")[0]
	// if proto != "tcp" || ip != "TODO(jerome):take from agent container struct"
	// Problem here: cannot check ip against the one we get from Docker by inspecting the container
	// as the two addresses do not match

	switch msg.GetType() {
	case "Handshake":
		{
			if _, found := s.agenthandshakes[msg.GetAgentId()]; found {
				return errors.New("ERROR: Received duplicate handshake from agent " + ag.String())
			}

			s.agenthandshakes[msg.GetAgentId()] = struct{}{}

			var handshake protocol.MessageHandshakeImp
			err = json.Unmarshal(msg.GetPayload(), &handshake)
			if err != nil {
				return errors.New("DispatchAgentMessage: Failed to unmarshal JSON agent handshake payload for agent " + msg.GetAgentId().String() + "; " + string(msg.GetPayload()))
			}

			ag, ok := ag.(agent.NetAgentInterface)
			if !ok {
				return errors.New("DispatchAgentMessage: Failed to cast agent to NetAgent during handshake for " + ag.String())
			}

			ag = ag.SetConn(msg.GetEmitterConn())
			s.setAgent(ag)

			utils.Debug("arena", "Received handshake from agent "+ag.String()+"; agent said \""+handshake.GetGreetings()+"\"")

			s.nbhandshaked++

			if s.nbhandshaked == s.getNbExpectedagents() {
				s.onAgentsReady()
			}

			// TODO(sven|jerome): handle some timeout here if all agents fail to handshake

			break
		}
	case "Mutation":
		{
			//break
			var mutations []protocol.MutationMessage
			err = json.Unmarshal(msg.GetPayload(), &mutations)
			if err != nil {
				return errors.New("DispatchAgentMessage: Failed to unmarshal JSON agent mutation payload for agent " + ag.String() + "; " + string(msg.GetPayload()))
			}

			mutationbatch := protocol.StateMutationBatch{
				AgentId:   ag.GetId(),
				Mutations: mutations,
			}

			s.PushMutationBatch(mutationbatch)

			break
		}
	default:
		{
			return errors.New("DispatchAgentMessage: Unknown message type" + msg.GetType())
		}
	}

	return nil
}

/* </implementing protocol.CommunicatorDispatcherInterface> */
