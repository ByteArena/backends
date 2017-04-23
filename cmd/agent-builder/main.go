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
	gitRepoUrl := os.Args[1]

	name := config.HashGitRepoName(gitRepoUrl)
	dir := cloneRepo(gitRepoUrl, name)

	buildImage(dir, name)
}

func buildImage(absBuildDir string, name string) {

	launchBuildProcess(
		name,
		getAbsoluteDir("build.sh"),
		absBuildDir,
	)
}

func cloneRepo(url string, hash string) string {
	/*
		authmethod, err := ssh.NewSSHAgentAuth("git") // git@github.com:...
		if err != nil {
			log.Panicln("Erreur sur NewSSHAgentAuth", err)
		}

		_, err = git.PlainClone("/tmp/"+dir, false, &git.CloneOptions{
			URL: url,
			//Auth:     transport,
			Progress: os.Stdout,
			Auth:     authmethod,
		})
	*/

	dir := "/tmp/" + hash
	err := os.RemoveAll(dir)

	cmd := exec.Command(getAbsoluteDir("clone.sh"), url, dir)

	cmdOut, err := cmd.Output()
	if err != nil {
		log.Panicln("Error running command: ", err, string(cmdOut))
	}

	out := string(cmdOut)

	log.Println("out", out)

	if err != nil {
		log.Panicln(err)
	}

	return dir
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
