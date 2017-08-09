package types

type ContestantType struct {
	Id              string               `json:"id"`
	Agent           *AgentType           `json:"agent"`
	Game            *GameType            `json:"game"`
	EnrolledAt      string               `json:"enrolledAt"`
	Deployment      *AgentDeploymentType `json:"deployment"`
	ContestantLogId string               `json:"contestantLogId"`
}
