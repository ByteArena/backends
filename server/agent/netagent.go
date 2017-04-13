package agent

import (
	"encoding/json"
	"net"

	"github.com/netgusto/bytearena/server/protocol"
	"github.com/netgusto/bytearena/server/state"
)

type NetAgent interface {
	Agent
	SetAddr(addr net.Addr) NetAgent
	GetAddr() net.Addr
}

type NetAgentImp struct {
	AgentImp
	addr net.Addr
}

func MakeNetAgentImp() NetAgentImp {
	return NetAgentImp{
		AgentImp: MakeAgentImp(),
	}
}

func (agent NetAgentImp) String() string {
	return "<NetAgentImp(" + agent.GetId().String() + ")>"
}

func (agent NetAgentImp) SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) {
	perceptionjson, _ := json.Marshal(perception)
	message := []byte("{\"Method\": \"tick\", \"Arguments\": [0," + string(perceptionjson) + "]}\n") // TODO: remove 0 (ex turn)
	comm.NetSend(message, agent.GetAddr())
}

func (agent NetAgentImp) SetAddr(addr net.Addr) NetAgent {
	agent.addr = addr
	return agent
}

func (agent NetAgentImp) GetAddr() net.Addr {
	return agent.addr
}
