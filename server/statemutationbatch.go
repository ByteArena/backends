package server

type StateMutationBatch struct {
	Turn      uint32
	Agent     *Agent
	Mutations []StateMutation
}
