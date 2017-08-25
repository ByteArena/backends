package protocol

type DatabaseInterface interface {
	Connect(connURI string) error
	ActivateDebug()
	Close()
	Migrate()
	CreateTables()
	FindUserByUsername(username string) (User, error)
	FindUserByEmail(email string) (User, error)
	FindRepository(user User, reponame string) (GitRepository, error)
	FindRepositoryById(id string) (GitRepository, error)
	FindPublicKeyByFingerprint(publickey string) (GitPublicKey, error)
	CreateUser(user User) error
	CreateRepository(repo GitRepository) error
	CreatePublicKey(key GitPublicKey) error
	DeleteRepository(repo GitRepository) error
	InjectFixtures(agentBuilderPublicKey string)
}
