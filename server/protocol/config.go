package protocol

type Config struct {
	Cmd string
}

type FileConfigWrapper struct {
	Agents []Config
}
