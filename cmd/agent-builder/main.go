package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/ttacon/chalk"

	"github.com/bytearena/bytearena/utils"
)

var CHANNEL = "agent"
var TOPIC = "repo.pushed"

type onMessageStruct struct {
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
	Topic     string          `json:"topic"`
	Channel   string          `json:"channel"`
}

func throwIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func buildAndDeploy(username string, repo string, gitHost string, registryHost string) {

	fqRepo := username + "/" + repo
	gitRepoURL := "git@" + gitHost + ":" + fqRepo + ".git"

	imageName := fqRepo
	dir := cloneRepo(gitRepoURL, imageName)
	buildImage(dir, imageName)
	deployImage(imageName, "latest", registryHost, 5000)
}

func main() {

	mqHost := os.Getenv("MQ_HOST")
	utils.Assert(mqHost != "", "Error: missing MQ_HOST env param")

	registryHost := os.Getenv("REGISTRY_HOST")
	utils.Assert(registryHost != "", "Error: missing REGISTRY_HOST env param")

	gitHost := os.Getenv("GIT_HOST")
	utils.Assert(gitHost != "", "Error: missing GIT_HOST env param")

	port := os.Getenv("PORT")
	utils.Assert(port != "", "Error: missing PORT env param")
	_, err := strconv.Atoi(port)
	utils.Check(err, "Error: PORT shoud be an int")

	listen(mqHost, gitHost, registryHost)
}

func listen(host string, gitHost string, registryHost string) {
	dialer := websocket.DefaultDialer

	conn, _, err := dialer.Dial("ws://"+host, http.Header{})

	utils.Check(err, "Error: cannot connect to host "+host)

	err = conn.WriteJSON(struct {
		Action  string `json:"action"`
		Channel string `json:"channel"`
		Topic   string `json:"topic"`
	}{
		"sub",
		CHANNEL,
		TOPIC,
	})

	utils.Check(err, "Error: cannot subscribe to message broker")

	for {
		_, rawData, err := conn.ReadMessage()
		utils.Check(err, "Received invalid message")

		var message onMessageStruct

		err = json.Unmarshal(rawData, &message)
		utils.Check(err, "Received invalid message")

		utils.Assert(
			message.Channel == CHANNEL && message.Topic == TOPIC,
			"unexpected message type, got "+message.Channel+":"+message.Topic,
		)

		var payload struct {
			Username string `json:"username"`
			Repo     string `json:"repo"`
		}

		err = json.Unmarshal(message.Data, &payload)
		utils.Check(err, "Received invalid payload")

		buildAndDeploy(payload.Username, payload.Repo, gitHost, registryHost)

		log.Println("Build successful")
	}
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

	// privatekey := "/root/git_admin_key_private"

	// sshbin, err := exec.LookPath("ssh")
	// if err != nil {
	// 	log.Fatal("Error: ssh not found in $PATH !")
	// }
	// cmd.Env = []string{fmt.Sprintf("GIT_SSH_COMMAND=\"%s\" -i \"%s\" -o \"StrictHostKeyChecking=no\"", sshbin, privatekey)}

	stdoutStderr, err := cmd.CombinedOutput()
	utils.Check(err, "Error running command: "+string(stdoutStderr))
	log.Println(fmt.Sprintf("%s%s%s", chalk.Yellow, stdoutStderr, chalk.Reset))

	return dir
}
