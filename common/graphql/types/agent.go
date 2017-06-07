package types

type AgentType struct {
	Id    string          `json:"id"`
	Name  string          `json:"name"`
	Repo  string          `json:"repo"`
	Owner UserType        `json:"owner"`
	Image DockerImageType `json:"image"`
}
