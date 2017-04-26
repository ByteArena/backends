package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/netgusto/bytearena/server/config"
	"github.com/netgusto/bytearena/utils"
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
		utils.GetAbsoluteDir("build.sh"),
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

	cmd := exec.Command(utils.GetAbsoluteDir("clone.sh"), url, dir)

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

	cmdOut, err = cmd.Output()
	utils.Check(err, "Error running command: "+string(cmdOut))

	out := string(cmdOut)
	log.Println("out", out)

	// Deploy

	cmd = exec.Command(
		utils.GetAbsoluteDir("deploy.sh"),
		name,
		"127.0.0.1:5000",
		"latest",
	)

	cmdOut, err = cmd.Output()
	utils.Check(err, "Error running command: "+string(cmdOut))

	out = string(cmdOut)
	log.Println("out", out)
}
