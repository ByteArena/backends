package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ttacon/chalk"

	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/utils"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	// handling signals
	go func() {
		<-common.SignalHandler()
		log.Println("RECEIVED SHUTDOWN SIGNAL; closing.")
		os.Exit(1)
	}()

	repoURL := flag.String("repourl", "", "URL of the git repository to build")
	registryHost := flag.String("registry", "registry.net.bytearena.com", "Base URL of the docker registry where to push image")
	imageName := flag.String("imagename", "", "Name of the image on the docker registry; example johndoe/happybot")

	flag.Parse()

	if *repoURL == "" {
		panic("Git repo URL is mandatory; you can specify it using the `--repourl` flag.")
	}

	if *registryHost == "" {
		panic("Docker registry Host is mandatory; you can specify it using the `--registry` flag.")
	}

	if *imageName == "" {
		panic("Docker image name is mandatory; you can specify it using the `--imagename` flag.")
	}

	if err := pingRegistry(*registryHost); err != nil {
		panic("Docker registry is unreachable; tried " + *registryHost)
	}

	buildAndDeploy(*repoURL, *registryHost, *imageName)
}

func pingRegistry(host string) error {
	resp, err := http.Get("http://" + host + "/v2/")

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Cannot ping registry")
	}

	return nil
}

func buildAndDeploy(cloneurl, registryHost, imageName string) {

	utils.Debug("agent-builder", "build and deploy image: "+imageName)

	err, dir := cloneRepo(cloneurl, imageName)

	if err == nil {
		err = buildImage(dir, imageName)

		if err == nil {
			deployImage(imageName, "latest", registryHost)
		} else {
			utils.Debug("error", err.Error())
		}

	} else {
		utils.Debug("error", err.Error())
	}
}

func buildImage(absBuildDir string, name string) error {
	utils.Debug("agent-builder", "Building agent")

	dockerbin, err := exec.LookPath("docker")
	utils.Check(err, "Error: docker command not found in path")

	cmd := exec.Command(
		dockerbin, "build", "-t",
		name,
		absBuildDir,
	)
	cmd.Env = nil

	stdoutStderr, err := cmd.CombinedOutput()

	if err != nil {
		return errors.New("Error running command: " + string(stdoutStderr))
	}

	log.Println(fmt.Sprintf("%s%s%s", chalk.Blue, stdoutStderr, chalk.Reset))

	return nil
}

func deployImage(name string, imageVersion string, registryhost string) {
	utils.Debug("agent-builder", "Deploying to docker registry")

	dockerbin, err := exec.LookPath("docker")
	utils.Check(err, "Error: docker command not found in path")

	imageurl := registryhost + "/" + name + ":" + imageVersion

	// Tag
	cmd := exec.Command(
		dockerbin, "tag",
		name,
		imageurl,
	)
	cmd.Env = nil
	stdoutStderr, err := cmd.CombinedOutput()
	utils.Check(err, "Error running TAG command: "+string(stdoutStderr))
	log.Println(fmt.Sprintf("%s%s%s", chalk.Yellow, stdoutStderr, chalk.Reset))

	// Push to remote registry
	cmd = exec.Command(
		dockerbin, "push",
		imageurl,
	)
	cmd.Env = nil

	stdoutStderr, err = cmd.CombinedOutput()
	utils.Check(err, "Error running PUSH command: "+string(stdoutStderr))
	log.Println(fmt.Sprintf("%s%s%s", chalk.Yellow, stdoutStderr, chalk.Reset))
}

func cloneRepo(url string, hash string) (error, string) {

	gitbin, err := exec.LookPath("git")
	utils.Check(err, "Error: git not found in $PATH")

	dir := "/tmp/" + hash
	os.RemoveAll(dir)
	utils.Debug("agent-builder", "Cloning "+url+" into "+dir)

	cmd := exec.Command(
		gitbin,
		"clone", "-b", "master",
		url,
		dir,
	)

	privatekey := "/root/git_admin_key_private"

	sshbin, err := exec.LookPath("ssh")
	if err != nil {
		log.Fatal("Error: ssh not found in $PATH")
	}
	cmd.Env = []string{fmt.Sprintf("GIT_SSH_COMMAND=\"%s\" -i \"%s\" -o \"StrictHostKeyChecking=no\"", sshbin, privatekey)}

	stdoutStderr, err := cmd.CombinedOutput()

	if err != nil {
		return errors.New("Error running command: " + string(stdoutStderr)), ""
	}

	return nil, dir
}
