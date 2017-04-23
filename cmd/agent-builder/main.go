package main

import (
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/kardianos/osext"
	// "github.com/netgusto/bytearena/server/config"
)

func throwIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	filename := os.Args[1]
	// config := config.LoadAgentConfig(filename)

	buildImage(path.Dir(filename))
}

func buildImage(buildDir string) {

	launchBuildProcess(
		"bytearena_bar",
		getAbsoluteDir("build.sh"),
		getAbsoluteDir(buildDir),
	)
}

func launchBuildProcess(name string, bin string, buildDir string) {
	cmd := exec.Command(bin, name, buildDir)

	var (
		cmdOut []byte
		err    error
	)

	if cmdOut, err = cmd.Output(); err != nil {
		log.Panicln("Error running command: ", err, string(cmdOut))
	}

	out := string(cmdOut)

	log.Println("out", out)

	// Deploy

	cmd = exec.Command(
		getAbsoluteDir("deploy.sh"),
		name,
		"127.0.0.1:5000",
		"latest",
	)

	if cmdOut, err = cmd.Output(); err != nil {
		log.Panicln("Error running command: ", err, string(cmdOut))
	}

	out = string(cmdOut)

	log.Println("out", out)
}

func getAbsoluteDir(relative string) string {

	exfolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	return path.Join(exfolder, relative)
}
