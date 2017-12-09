package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/bytearena/core/common"
	"github.com/bytearena/core/common/dockerfile"
	"github.com/bytearena/core/common/utils"
)

func msgOut(msg string) {
	msgOutNoExit(msg)
	os.Exit(1)
}

func msgOutNoExit(msg string) {
	fmt.Println("🛑  " + msg)
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
	deploymentid := flag.String("deploymentid", "", "Deploiement identifier")

	flag.Parse()

	if *repoURL == "" {
		msgOut("RECEIVED SHUTDOWN SIGNAL; closing.")
	}

	if *deploymentid == "" {
		msgOut("Missing deploymentid")
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

	if err := buildAndDeploy(*repoURL, *registryHost, *imageName, *deploymentid); err != nil {
		msgOut("Could not build/deploy agent; " + err.Error())
		os.Exit(1)
	}
}

func pingRegistry(host string) error {
	resp, err := http.Get("https://" + host + "/v2/")

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Cannot ping registry")
	}

	return nil
}

func buildAndDeploy(cloneurl, registryHost, imageName, deploymentid string) error {

	welcomeBanner()

	dir := cloneRepo(cloneurl, fmt.Sprintf("%s-%d", imageName, time.Now().UnixNano()))

	buildImage(dir, imageName, registryHost)

	deployImage(imageName, deploymentid, registryHost)

	successBanner()

	return nil
}

func assertAgentCodeIsLegit(absBuildDir, registryHost string) {

	maxDockerfileSizeInByte := 8192
	agentImageNamespace := "bytearena/agent/"
	systemImageNamespace := "bytearena/"
	registryImageNamespace := registryHost

	dockerfilePath := absBuildDir + "/Dockerfile"

	// on vérifie que le chemin contient bien un Dockerfile
	dockerfileStat, err := os.Stat(dockerfilePath)

	if os.IsNotExist(err) {
		msgOut("Error: Your agent code does not contain the required Dockerfile.")
	}

	if !dockerfileStat.Mode().IsRegular() {
		msgOut("Error: Your agent's Dockerfile is not a valid file.")
	}

	if dockerfileStat.Size() > int64(maxDockerfileSizeInByte) {
		msgOut("Error: Your agent's Dockerfile is bigger than the limit of " + strconv.Itoa(maxDockerfileSizeInByte) + " bytes.")
	}

	dockerfilePointer, err := os.OpenFile(dockerfilePath, os.O_RDONLY, 0666)
	if err != nil {
		msgOut("Error: Could not read your agent's Dockerfile.")
	}
	defer dockerfilePointer.Close()

	dockerfileContent := make([]byte, maxDockerfileSizeInByte)
	size, err := dockerfilePointer.Read(dockerfileContent)
	if err != nil {
		msgOut("Error: Could not read your agent's Dockerfile.")
	}

	dockerfileContent = dockerfileContent[:size]

	// on vérifie que le Dockerfile ne contient que des FROM légitimes
	froms, err := dockerfile.DockerfileParserGetFroms(bytes.NewReader(dockerfileContent))
	if err != nil {
		msgOut("Error: Your agent's Dockerfile cannot be parsed.")
	}

	for _, from := range froms {
		if strings.HasPrefix(from, agentImageNamespace) {
			msgOut("Error: Your agent Dockerfile cannot extend images from the namespace " + agentImageNamespace)
		}

		if strings.HasPrefix(from, systemImageNamespace) {
			msgOut("Error: Your agent Dockerfile cannot extend images from the namespace " + systemImageNamespace)
		}

		if strings.HasPrefix(from, registryImageNamespace) {
			msgOut("Error: Your agent Dockerfile cannot extend images from the namespace " + registryImageNamespace)
		}
	}

	forbiddenInstructions, err := dockerfile.DockerfileFindForbiddenInstructions(bytes.NewReader(dockerfileContent))

	if err != nil {
		msgOut("Error: Your agent's Dockerfile cannot be parsed.")
	}

	for name, _ := range forbiddenInstructions {
		msgOutNoExit("Error: forbidden instruction in Dockerfile: `" + name.String() + "`.")
	}

	if len(forbiddenInstructions) > 0 {
		msgOut("Agent was not built because the Dockerfile is not valid.")
	}
}

func buildImage(absBuildDir, name, registryHost string) {

	assertAgentCodeIsLegit(absBuildDir, registryHost)

	dockerbin, err := exec.LookPath("docker")
	if err != nil {
		msgOut("Error: docker command not found in path")
	}

	dockerBuildMemoryLimit := utils.GetenvOrDefault("DOCKER_BUILD_MEMORY_LIMIT", "100m")
	dockerBuildSwapLimit := utils.GetenvOrDefault("DOCKER_BUILD_SWAP_LIMIT", "100m")
	dockerBuildNetwork := utils.GetenvOrDefault("DOCKER_BUILD_NETWORK", "bridge")
	dockerBuildNoCache := utils.GetenvOrDefault("DOCKER_BUILD_NO_CACHE", "true")
	dockerBuildCpuPeriod := utils.GetenvOrDefault("DOCKER_BUILD_CPU_PERIOD", "5000")

	dockargs := []string{
		"build",
		"-t", name,
		"--memory", dockerBuildMemoryLimit,
		"--memory-swap", dockerBuildSwapLimit,
		"--network", dockerBuildNetwork,
		"--cpu-period", dockerBuildCpuPeriod,
	}

	if dockerBuildNoCache == "true" {
		dockargs = append(dockargs, "--no-cache")
	}

	dockargs = append(dockargs, absBuildDir)

	cmd := exec.Command(
		dockerbin,
		dockargs...,
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
