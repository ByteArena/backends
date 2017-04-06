package utils

import (
	"strconv"

	uuid "github.com/satori/go.uuid"
)

type Tickturn struct {
	seq int
	id  uuid.UUID
}

func (turn Tickturn) String() string {
	return "<TickTurn(" + strconv.Itoa(int(turn.seq)) + ")>"
}

func (turn Tickturn) Next() Tickturn {
	return Tickturn{
		seq: turn.seq + 1,
		id:  uuid.NewV4(),
	}
}

func (turn Tickturn) GetSeq() int {
	return turn.seq
}
