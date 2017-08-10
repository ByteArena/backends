package types

type AgentType struct {
	Id            string                `json:"id"`
	Name          string                `json:"name"`
	Title         string                `json:"title"`
	Image         *DockerImageType      `json:"image"`
	GitRepository *GitRepositoryType    `json:"gitRepository"`
	Owner         *UserType             `json:"owner"`
	Contestants   []ContestantType      `json:"contestants"`
	Deployments   []AgentDeploymentType `json:"deployments"`
}
