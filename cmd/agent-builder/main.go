package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/ttacon/chalk"

	"github.com/bytearena/bytearena/server/config"
	"github.com/bytearena/bytearena/utils"
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
	deployImage(name, "latest", "127.0.0.1", 5000)
}

func buildImage(absBuildDir string, name string) {

	log.Println(fmt.Sprintf("%sBuilding agent%s", chalk.Blue, chalk.Reset))

	dockerbin, err := exec.LookPath("docker")
	utils.Check(err, "Error: docker command not found in path")

	cmd := exec.Command(
		dockerbin, "build", "-t",
		name,
		absBuildDir,
	)
	cmd.Env = nil

	stdoutStderr, err := cmd.CombinedOutput()
	utils.Check(err, "Error running command: "+string(stdoutStderr))
	log.Println(fmt.Sprintf("%s%s%s", chalk.Blue, stdoutStderr, chalk.Reset))
}

func deployImage(name string, tag string, registryhost string, registryport int) {

	log.Println(fmt.Sprintf("%sDeploying to docker registry%s", chalk.Yellow, chalk.Reset))

	dockerbin, err := exec.LookPath("docker")
	utils.Check(err, "Error: docker command not found in path")

	imageurl := registryhost + ":" + strconv.Itoa(registryport) + "/" + name + ":" + tag

	// Tag
	cmd := exec.Command(
		dockerbin, "tag",
		name,
		imageurl,
	)
	cmd.Env = nil

	// Push to remote registry
	cmd = exec.Command(
		dockerbin, "push",
		imageurl,
	)
	cmd.Env = nil

	stdoutStderr, err := cmd.CombinedOutput()
	utils.Check(err, "Error running command: "+string(stdoutStderr))
	log.Println(fmt.Sprintf("%s%s%s", chalk.Yellow, stdoutStderr, chalk.Reset))
}

func cloneRepo(url string, hash string) string {

	gitbin, err := exec.LookPath("git")
	utils.Check(err, "Error: git not found in $PATH !")

	dir := "/tmp/" + hash
	os.RemoveAll(dir)
	log.Println(fmt.Sprintf("%sCloning %s into %s%s", chalk.Yellow, url, dir, chalk.Reset))

	cmd := exec.Command(
		gitbin,
		"clone", "-b", "master",
		url,
		dir,
	)

	/*
		privatekey := "/Users/jerome/.ssh/bytearenaserver"
		sshbin, err := exec.LookPath("ssh")
		if err != nil {
			log.Fatal("Error: ssh not found in $PATH !")
		}
		cmd.Env = []string{fmt.Sprintf("GIT_SSH_COMMAND=\"%s\" -i \"%s\"", sshbin, privatekey)}
	*/

	stdoutStderr, err := cmd.CombinedOutput()
	utils.Check(err, "Error running command: "+string(stdoutStderr))
	log.Println(fmt.Sprintf("%s%s%s", chalk.Yellow, stdoutStderr, chalk.Reset))

	return dir
}
