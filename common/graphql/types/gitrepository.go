package types

type GitRepositoryType struct {
	CloneURL string `json:"cloneurl"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Ref      string `json:"ref"`
}
