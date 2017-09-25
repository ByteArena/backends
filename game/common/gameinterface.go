package common

import (
	"github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

type GameEventSubscription int32

type GameInterface interface {
	ImplementsGameInterface()
	Subscribe(event string, cbk func(data interface{})) GameEventSubscription
	Unsubscribe(subscription GameEventSubscription)
	Step(dt float64, mutations []types.AgentMutationBatch)
	NewEntityAgent(pos vector.Vector2) *ecs.Entity
	ComputeAgentPerception(arenamap *mapcontainer.MapContainer, entityid ecs.EntityID) []byte
	ProduceVizMessageJson() []byte
}
