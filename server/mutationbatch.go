package main

type MutationBatch struct {
	Turn      uint32
	Agent     *Agent
	Mutations []StateMutation
}
