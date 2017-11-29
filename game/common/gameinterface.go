package common

import (
	"github.com/bytearena/bytearena/arenaserver/types"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

type GameEventSubscription int32

type GameInterface interface {
	ImplementsGameInterface()
	Subscribe(event string, cbk func(data interface{})) GameEventSubscription
	Unsubscribe(subscription GameEventSubscription)
	Step(tickturn int, dt float64, mutations []types.AgentMutationBatch)
	NewEntityAgent(contestant commontypes.Contestant, pos vector.Vector2) *ecs.Entity

	GetAgentPerception(entityid ecs.EntityID) []byte
	GetAgentWelcome(entityid ecs.EntityID) []byte

	GetVizFrameJson() []byte
}
