package deathmatch

import "github.com/bytearena/ecs"

type eventType uint8

// Prefer use lightweight representation of constants for the future transport
const (
	EVENT_PROJECTILE_KILLED_ENTITY eventType = 1 << iota
)

type Event struct {
	EventType    eventType
	TargetEntity *ecs.Entity
}

type Events []Event

func MakeEmtpyEvents() Events {
	return make(Events, 0)
}

func NewEventOfType(eventType eventType, targetEntity *ecs.Entity) *Event {
	return &Event{
		EventType:    eventType,
		TargetEntity: targetEntity,
	}
}
