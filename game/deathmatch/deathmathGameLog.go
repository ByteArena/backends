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
	entries []LogEntry
}

func NewDeathmatchGameLog() *DeathmatchGameLog {
	return &DeathmatchGameLog{
		entries: make([]LogEntry, 0),
	}
}

func MakeLogEntryOfType(eventType eventType, targetEntity *ecs.Entity) LogEntry {
	return LogEntry{
		EventType:    eventType,
		TargetEntity: targetEntity,
	}
}

func (l *DeathmatchGameLog) AddEntry(entry LogEntry) {
	l.entries = append(l.entries, entry)
}
