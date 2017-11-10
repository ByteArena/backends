package deathmatch

import "github.com/bytearena/ecs"

// Prefer use lightweight representation of constants for the future transport
const (
	EVENT_PROJECTILE_KILLED_ENTITY eventType = 1 << iota
)

type eventType uint8

type LogEntry struct {
	EventType    eventType
	TargetEntity *ecs.Entity
}

type DeathmatchGameLog struct {
	entries map[*ecs.Entity][]LogEntry
}

func NewDeathmatchGameLog() *DeathmatchGameLog {
	return &DeathmatchGameLog{
		entries: make(map[*ecs.Entity][]LogEntry, 0),
	}
}

func MakeLogEntryOfType(eventType eventType, targetEntity *ecs.Entity) LogEntry {
	return LogEntry{
		EventType:    eventType,
		TargetEntity: targetEntity,
	}
}

func (l *DeathmatchGameLog) AddEntryForEntity(key *ecs.Entity, entry LogEntry) {

	// Create the entity's mailbox if needed
	if _, hasMailbox := l.entries[key]; !hasMailbox {
		l.entries[key] = make([]LogEntry, 0)
	}

	// Append to its mailbox this new event
	l.entries[key] = append(l.entries[key], entry)
}
