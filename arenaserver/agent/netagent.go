package agent

import (
	"encoding/json"
	"net"

	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/arenaserver/state"
)

type NetAgent interface {
	Agent
	SetConn(conn net.Conn) NetAgent
	GetConn() net.Conn
}

type NetAgentImp struct {
	AgentImp
	conn net.Conn
}

func MakeNetAgentImp() NetAgentImp {
	return NetAgentImp{
		AgentImp: MakeAgentImp(),
	}
}

func (agent NetAgentImp) String() string {
	return "<NetAgentImp(" + agent.GetId().String() + ")>"
}

func (agent NetAgentImp) SetPerception(perception state.Perception, comm protocol.AgentCommunicator) error {
	perceptionjson, _ := json.Marshal(perception)
	message := []byte("{\"Method\": \"tick\", \"Arguments\": [0," + string(perceptionjson) + "]}\n") // TODO(jerome): remove 0 (ex turn)
	return comm.NetSend(message, agent.GetConn())
}

func (agent NetAgentImp) SetConn(conn net.Conn) NetAgent {
	agent.conn = conn
	return agent
}

func (agent NetAgentImp) GetConn() net.Conn {
	return agent.conn
}
