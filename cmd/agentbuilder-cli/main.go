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
		log.Println("RECEIVED SHUTDOWN SIGNAL; closing.")
		os.Exit(1)
	}

	if *registryHost == "" {
		log.Println("Docker registry Host is mandatory; you can specify it using the `--registry` flag.")
		os.Exit(1)
	}

	if *imageName == "" {
		log.Println("Docker image name is mandatory; you can specify it using the `--imagename` flag.")
		os.Exit(1)
	}

	if err := pingRegistry(*registryHost); err != nil {
		log.Println("Docker registry is unreachable; tried " + *registryHost)
		os.Exit(1)
	}

	if err := buildAndDeploy(*repoURL, *registryHost, *imageName); err != nil {
		log.Println("Could not build/deploy agent; " + err.Error())
		os.Exit(1)
	}
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

func buildAndDeploy(cloneurl, registryHost, imageName string) error {

	utils.Debug("agentbuilder-cli", "build and deploy image: "+imageName)

	err, dir := cloneRepo(cloneurl, fmt.Sprintf("%s-%d", imageName, time.Now().UnixNano()))

	if err == nil {
		err = buildImage(dir, imageName)

		if err == nil {
			deployImage(imageName, "latest", registryHost)
		} else {
			utils.Debug("error", err.Error())
			return err
		}

	} else {
		utils.Debug("error", err.Error())
		return err
	}

	return nil
}

func buildImage(absBuildDir string, name string) error {
	utils.Debug("agentbuilder-cli", "Building agent")

	dockerbin, err := exec.LookPath("docker")
	utils.Check(err, "Error: docker command not found in path")

	dockerBuildMemoryLimit := os.Getenv("DOCKER_BUILD_MEMORY_LIMIT")
	dockerBuildNetwork := os.Getenv("DOCKER_BUILD_NETWORK")
	dockerBuildNoCache := os.Getenv("DOCKER_BUILD_NO_CACHE")
	dockerBuildCpuPeriod := os.Getenv("DOCKER_BUILD_CPU_PERIOD")

	cmd := exec.Command(
		dockerbin, "build",
		"-t", name,
		"--memory", dockerBuildMemoryLimit,
		"--network", dockerBuildNetwork,
		"--no-cache", dockerBuildNoCache,
		"--cpu-period", dockerBuildCpuPeriod,
		absBuildDir,
	)
	cmd.Env = nil
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		log.Println("Error: could not build image; " + err.Error())
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		log.Println("Error: could not build image; " + err.Error())
		os.Exit(1)
	}

	return nil
}

func deployImage(name string, imageVersion string, registryhost string) {
	utils.Debug("agentbuilder-cli", "Deploying to docker registry")

	dockerbin, err := exec.LookPath("docker")
	if err != nil {
		log.Println("Error: docker command not found in path" + err.Error())
		os.Exit(1)
	}

	imageurl := registryhost + "/" + name + ":" + imageVersion

	// Tag
	cmd := exec.Command(
		dockerbin, "tag",
		name,
		imageurl,
	)
	cmd.Env = nil
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		log.Println("Error: could not tag image; " + err.Error())
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		log.Println("Error: could not tag image; " + err.Error())
		os.Exit(1)
	}

	// Push to remote registry
	cmd2 := exec.Command(
		dockerbin, "push",
		imageurl,
	)
	cmd2.Env = nil
	cmd2.Stdin = os.Stdin
	cmd2.Stdout = os.Stdout

	if err := cmd2.Start(); err != nil {
		log.Println("Error: could not push image to registry; " + err.Error())
		os.Exit(1)
	}

	if err := cmd2.Wait(); err != nil {
		log.Println("Error: could not push image to registry; " + err.Error())
		os.Exit(1)
	}
}

func cloneRepo(url string, hash string) (error, string) {

	gitbin, err := exec.LookPath("git")
	if err != nil {
		log.Println("Error: git not found in $PATH")
		os.Exit(1)
	}

	dir := "/tmp/" + hash
	os.RemoveAll(dir)
	utils.Debug("agentbuilder-cli", "Cloning "+url+" into "+dir)

	cmd := exec.Command(
		gitbin,
		"clone", "-b", "master",
		url,
		dir,
	)

	sshbin, err := exec.LookPath("ssh")
	if err != nil {
		log.Println("Error: ssh not found in $PATH")
		os.Exit(1)
	}
	cmd.Env = []string{fmt.Sprintf("GIT_SSH_COMMAND=\"%s\" -o \"StrictHostKeyChecking=no\"", sshbin)}

	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New("Error running command: " + string(stdoutStderr)), ""
	}

	return nil, dir
}
