package server

type StateMutationBatch struct {
	Turn      tickturn
	Agent     *Agent
	Mutations []StateMutation
}
