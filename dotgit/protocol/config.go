package protocol

type ConfigInterface interface {
	GetDatabaseURI() string
	GitRepositoriesPath() string
}
