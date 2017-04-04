package utils

import (
	"strconv"

	uuid "github.com/satori/go.uuid"
)

type Tickturn struct {
	seq uint32
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

func (turn Tickturn) GetSeq() uint32 {
	return turn.seq
}
