package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/ttacon/chalk"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func main() {

	mqHost := os.Getenv("MQ_HOST")
	utils.Assert(mqHost != "", "Error: missing MQ_HOST env param")

	registryHost := os.Getenv("REGISTRY_HOST")
	utils.Assert(registryHost != "", "Error: missing REGISTRY_HOST env param")

	gitHost := os.Getenv("GIT_HOST")
	utils.Assert(gitHost != "", "Error: missing GIT_HOST env param")

	brokerclient, err := mq.NewClient(mqHost)
	utils.Check(err, "ERROR: could not connect to messagebroker at "+string(mqHost))

	brokerclient.Subscribe("agent", "repo.pushed", func(msg mq.BrokerMessage) {
		onRepoPushedMessage(msg, registryHost)
	})

	StartHealthCheck(brokerclient, registryHost)
}

func onRepoPushedMessage(msg mq.BrokerMessage, registryHost string) {

	log.Println(string(msg.Data))

	var message types.MQMessage
	err := json.Unmarshal(msg.Data, &message)
	if err != nil {
		log.Println(err)
		log.Println("ERROR:agent Invalid MQMessage " + string(msg.Data))
		return
	}

	if message.Payload == nil {
		log.Println("ERROR:agent Invalid Payload in MQMessage")
		return
	}

	payload := (*message.Payload)

	username, ok := payload["username"].(string)
	if !ok {
		log.Println("ERROR: missing username in MQMessage")
		return
	}

	repo, ok := payload["repo"].(string)
	if !ok {
		log.Println("ERROR: missing repo in MQMessage")
		return
	}

	cloneurl, ok := payload["cloneurl"].(string)
	if !ok {
		log.Println("ERROR: missing cloneurl in MQMessage")
		return
	}

	buildAndDeploy(username, repo, cloneurl, registryHost)
}

func buildAndDeploy(username string, repo string, cloneurl string, registryHost string) {
	imageName := username + "/" + repo

	dir := cloneRepo(cloneurl, imageName)
	buildImage(dir, imageName)
	deployImage(imageName, "latest", registryHost, 5000)
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

func deployImage(name string, imageVersion string, registryhost string, registryport int) {

	log.Println(fmt.Sprintf("%sDeploying to docker registry%s", chalk.Yellow, chalk.Reset))

	dockerbin, err := exec.LookPath("docker")
	utils.Check(err, "Error: docker command not found in path")

	imageurl := registryhost + ":" + strconv.Itoa(registryport) + "/" + name + ":" + imageVersion

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

	privatekey := "/root/git_admin_key_private"

	sshbin, err := exec.LookPath("ssh")
	if err != nil {
		log.Fatal("Error: ssh not found in $PATH !")
	}
	cmd.Env = []string{fmt.Sprintf("GIT_SSH_COMMAND=\"%s\" -i \"%s\" -o \"StrictHostKeyChecking=no\"", sshbin, privatekey)}

	stdoutStderr, err := cmd.CombinedOutput()
	utils.Check(err, "Error running command: "+string(stdoutStderr))
	log.Println(fmt.Sprintf("%s%s%s", chalk.Yellow, stdoutStderr, chalk.Reset))

	return dir
}
