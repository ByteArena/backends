package protocol

import (
	serverprotocol "github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/utils/vector"
)

type AgentStateInterface interface {
	Clone() AgentStateInterface
	ProcessMutations(mutations []serverprotocol.MessageMutation) AgentStateInterface
	Update() AgentStateInterface

	GetOrientation() float64
	GetVelocity() vector.Vector2
	GetPosition() vector.Vector2
	GetRadius() float64
	GetVisionRadius() float64
	GetVisionAngle() float64

	MutationSteer(vec vector.Vector2) AgentStateInterface
}
