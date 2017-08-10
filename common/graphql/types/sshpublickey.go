package types

type SSHPublicKeyType struct {
	Owner       *UserType `json:"owner"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Key         string    `json:"key"`
	Fingerprint string    `json:"fingerprint"`
	Comment     string    `json:"comment"`
}
