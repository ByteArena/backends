package types

type AgentDeploymentType struct {
	Id             string `json:"id"`
	PushedAt       string `json:"pushedAt"`
	CommitSHA1     string `json:"commitSHA1"`
	CommitMessage  string `json:"commitMessage"`
	BuildStartedAt string `json:"buildStartedAt"`
	BuildEndedAt   string `json:"buildEndedAt"`
	BuildStatus    int    `json:"buildStatus"`
	BuildError     bool   `json:"buildError"`
	BuildLogId     string `json:"buildLogId"`
}

var AgentDeployBuildStatus = struct {
	Pending  int
	Building int
	Finished int
}{
	Pending:  0,
	Building: 1,
	Finished: 2,
}
