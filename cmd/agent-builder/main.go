package main

import (
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/kardianos/osext"
	"github.com/netgusto/bytearena/server/config"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

func throwIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	gitRepoUrl := os.Args[1]

	name := config.HashGitRepoName(gitRepoUrl)
	dir := cloneRepo(gitRepoUrl, name)

	buildImage(dir, name)
}

func buildImage(buildDir string, name string) {

	launchBuildProcess(
		name,
		getAbsoluteDir("build.sh"),
		getAbsoluteDir(buildDir),
	)
}

func cloneRepo(url string, dir string) string {
	_, err := git.PlainClone("/tmp/"+dir, false, &git.CloneOptions{
		URL:      url,
		Auth:     Transport.ssh,
		Progress: os.Stdout,
	})

	if err != nil {
		log.Panicln(err)
	}

	return ""
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
