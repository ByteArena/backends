package types

type AgentType struct {
	Id            string            `json:"id"`
	Name          string            `json:"name"`
	GitRepository GitRepositoryType `json:"gitrepository"`
	Owner         UserType          `json:"owner"`
	Image         DockerImageType   `json:"image"`
}
