package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/bytearena/bytearena/common"
)

func msgOut(msg string) {
	log.Println("🛑 " + msg)
	os.Exit(1)
}

func welcomeBanner() {
	fmt.Println("=== ")
	fmt.Println("=== 🤖  Welcome on Byte Arena Builder Bot !")
	fmt.Println("=== ")
	fmt.Println("")
}

func successBanner() {
	fmt.Println("")
	fmt.Println("=== ")
	fmt.Println("=== ✅  Your agent is deployed. Let'em know who's the best !")
	fmt.Println("=== ")
	fmt.Println("")
}

func main() {

	// handling signals
	go func() {
		<-common.SignalHandler()
		msgOut("RECEIVED SHUTDOWN SIGNAL; closing.")
	}()

	repoURL := flag.String("repourl", "", "URL of the git repository to build")
	registryHost := flag.String("registry", "registry.net.bytearena.com", "Base URL of the docker registry where to push image")
	imageName := flag.String("imagename", "", "Name of the image on the docker registry; example johndoe/happybot")

	flag.Parse()

	if *repoURL == "" {
		msgOut("RECEIVED SHUTDOWN SIGNAL; closing.")
	}

	if *registryHost == "" {
		msgOut("Docker registry Host is mandatory; you can specify it using the `--registry` flag.")
	}

	if *imageName == "" {
		msgOut("Docker image name is mandatory; you can specify it using the `--imagename` flag.")
	}

	if err := pingRegistry(*registryHost); err != nil {
		msgOut("Docker registry is unreachable; tried " + *registryHost)
	}

	if err := buildAndDeploy(*repoURL, *registryHost, *imageName); err != nil {
		msgOut("Could not build/deploy agent; " + err.Error())
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

	//utils.Debug("agentbuilder-cli", "build and deploy image: "+imageName)

	welcomeBanner()

	dir := cloneRepo(cloneurl, fmt.Sprintf("%s-%d", imageName, time.Now().UnixNano()))

	buildImage(dir, imageName)

	deployImage(imageName, "latest", registryHost)

	successBanner()

	return nil
}

func buildImage(absBuildDir string, name string) {
	//utils.Debug("agentbuilder-cli", "Building agent")

	dockerbin, err := exec.LookPath("docker")
	if err != nil {
		msgOut("Error: docker command not found in path")
	}

	cmd := exec.Command(
		dockerbin, "build", "-t",
		name,
		absBuildDir,
	)
	cmd.Env = nil
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		msgOut("Error: could not start building process")
	}

	if err := cmd.Wait(); err != nil {
		msgOut("Error: could not build image.\n\n" + err.Error())
	}
}

func deployImage(name string, imageVersion string, registryhost string) {
	//utils.Debug("agentbuilder-cli", "Deploying to docker registry")

	dockerbin, err := exec.LookPath("docker")
	if err != nil {
		msgOut("Error: docker command not found in path")
	}

	imageurl := registryhost + "/" + name + ":" + imageVersion

	// Tag
	cmd := exec.Command(
		dockerbin, "tag",
		name,
		imageurl,
	)
	cmd.Env = nil
	// cmd.Stdin = os.Stdin
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		msgOut("Error: could not tag image")
	}

	if err := cmd.Wait(); err != nil {
		msgOut("Error: could not tag image")
	}

	// Push to remote registry
	cmd2 := exec.Command(
		dockerbin, "push",
		imageurl,
	)
	cmd2.Env = nil
	// cmd2.Stdin = os.Stdin
	// cmd2.Stdout = os.Stdout
	// cmd2.Stderr = os.Stderr

	if err := cmd2.Start(); err != nil {
		msgOut("Error: could not push image to registry")
	}

	if err := cmd2.Wait(); err != nil {
		msgOut("Error: could not push image to registry")
	}
}

func cloneRepo(url string, hash string) string {

	gitbin, err := exec.LookPath("git")
	if err != nil {
		msgOut("Error: git not found in $PATH")
	}

	dir := "/tmp/" + hash
	os.RemoveAll(dir)
	//utils.Debug("agentbuilder-cli", "Cloning "+url+" into "+dir)

	cmd := exec.Command(
		gitbin,
		"clone", "-b", "master",
		url,
		dir,
	)

	sshbin, err := exec.LookPath("ssh")
	if err != nil {
		msgOut("Error: ssh not found in $PATH")
	}
	cmd.Env = []string{fmt.Sprintf("GIT_SSH_COMMAND=\"%s\" -o \"StrictHostKeyChecking=no\"", sshbin)}

	_, err = cmd.CombinedOutput()
	if err != nil {
		msgOut("Error: could not clone repository")
	}

	return dir
}
