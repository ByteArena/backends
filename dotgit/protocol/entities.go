package protocol

type User struct {
	ID              uint   `json:"-"`
	Username        string `json:"username"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	UniversalReader bool   `json:"-"`
	UniversalWriter bool   `json:"-"`
}

func (user User) String() string {
	return "<User(" + user.Username + ")>"
}

type GitRepository struct {
	ID       uint   `json:"-"`
	CloneURL string `json:"cloneurl"`
	Ref      string `json:"ref"`
	Name     string `json:"name"`
	Owner    User   `json:"-"`
	OwnerID  int    `json:"-"`
}

func (repo GitRepository) String() string {
	return "<GitRepository(" + repo.CloneURL + ")>"
}

type GitPublicKey struct {
	ID          uint   `json:"-"`
	Owner       User   `json:"-"`
	OwnerID     int    `json:"-"`
	KeyName     string `json:"keyname"`
	KeyType     string `json:"keytype"` // ssh-rsa
	Key         string `json:"key"`     // key
	Comment     string `json:"comment"` // comment
	Fingerprint string `json:"-"`
}

func (key GitPublicKey) String() string {
	return "<GitPublicKey(" + key.Owner.Username + "/" + key.KeyName + ")>"
}
