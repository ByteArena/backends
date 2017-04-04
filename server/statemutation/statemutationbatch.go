package statemutation

import (
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
)

type StateMutationBatch struct {
	Turn      utils.Tickturn
	Agent     uuid.UUID
	Mutations []StateMutation
}
