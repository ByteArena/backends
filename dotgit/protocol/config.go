package protocol

type Config interface {
	GetDatabaseURI() string
	GitRepositoriesPath() string
}
