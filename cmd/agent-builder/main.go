package main

import (
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/kardianos/osext"
	"github.com/netgusto/bytearena/server/config"
)

func throwIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	filename := os.Args[1]
	config := config.LoadAgentConfig(filename)

	switch config.Image {
	case "nodejs":
		buildImage(config.Image, path.Dir(filename))
	default:
		log.Panicln("Supported images are: 'node', " + config.Image + " given")
	}
}

func buildImage(buildPackName string, buildDir string) {

	launchBuildProcess(
		buildPackName,
		getAbsoluteDir("build.sh"),
		getAbsoluteDir(buildDir),
	)
}

func launchBuildProcess(buildPackName string, bin string, buildDir string) {
	cmd := exec.Command(bin, buildPackName, buildDir)

	var (
		cmdOut []byte
		err    error
	)

	if cmdOut, err = cmd.Output(); err != nil {
		log.Panicln("Error running command: ", err, string(cmdOut))
	}

	out := string(cmdOut)

	log.Println("out", out)
}

func getAbsoluteDir(relative string) string {

	exfolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	return path.Join(exfolder, relative)
}
