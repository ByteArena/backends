package agent

import (
	"net"
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

func (agent NetAgentImp) SetAddr(addr net.Addr) NetAgent {
	agent.addr = addr
	return agent
}

func (agent NetAgentImp) GetAddr() net.Addr {
	return agent.addr
}
