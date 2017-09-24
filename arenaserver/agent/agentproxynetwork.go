package agent

import (
	"encoding/json"
	"net"

	"github.com/bytearena/bytearena/arenaserver/protocol"
)

type AgentProxyNetworkInterface interface {
	AgentProxyInterface
	SetConn(conn net.Conn) AgentProxyNetworkInterface
	GetConn() net.Conn
}

type AgentProxyNetwork struct {
	AgentProxyGeneric
	conn net.Conn
}

func MakeAgentProxyNetwork() AgentProxyNetwork {
	return AgentProxyNetwork{
		AgentProxyGeneric: MakeAgentProxyGeneric(),
	}
}

func (agent AgentProxyNetwork) String() string {
	return "<NetAgentImp(" + agent.GetProxyUUID().String() + ")>"
}

func (agent AgentProxyNetwork) SetPerception(perception protocol.AgentPerception, comm protocol.AgentCommunicatorInterface) error {
	perceptionjson, _ := json.Marshal(perception)
	message := []byte("{\"Method\": \"tick\", \"Arguments\": [0," + string(perceptionjson) + "]}\n") // TODO(jerome): remove 0 (ex turn)
	return comm.NetSend(message, agent.GetConn())
}

func (agent AgentProxyNetwork) SetConn(conn net.Conn) AgentProxyNetworkInterface {
	agent.conn = conn
	return agent
}

func (agent AgentProxyNetwork) GetConn() net.Conn {
	return agent.conn
}