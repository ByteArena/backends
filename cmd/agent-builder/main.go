package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/ttacon/chalk"

	"github.com/bytearena/bytearena/common"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

func main() {

	env := os.Getenv("ENV")

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

	// handling signals
	if env == "prod" {
		hc := NewHealthCheck(brokerclient, registryHost)
		hc.Start()

		<-common.SignalHandler()
		utils.Debug("sighandler", "RECEIVED SHUTDOWN SIGNAL; closing.")
		hc.Stop()
	} else {
		// block
		<-common.SignalHandler()
	}
}

func onRepoPushedMessage(msg mq.BrokerMessage, registryHost string) {
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

	utils.Debug("agent-builder", "build and deploy image: "+imageName)

	err, dir := cloneRepo(cloneurl, imageName)

	if err == nil {
		err = buildImage(dir, imageName)

		if err == nil {
			deployImage(imageName, "latest", registryHost, 5000)
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

func deployImage(name string, imageVersion string, registryhost string, registryport int) {
	utils.Debug("agent-builder", "Deploying to docker registry")

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
