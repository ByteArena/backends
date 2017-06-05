package types

type AgentType struct {
	Id    string          `json:"id"`
	Name  string          `json:"name"`
	Repo  string          `json:"repo"`
	Owner OwnerType       `json:"owner"`
	Image DockerImageType `json:"image"`
}
