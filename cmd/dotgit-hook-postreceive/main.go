package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bytearena/bytearena/common/graphql"
	gqltypes "github.com/bytearena/bytearena/common/graphql/types"
)

const createDeploymentMutation = `
mutation($agentDeployment: AgentDeploymentInputCreate!) {
	createAgentDeployment(agentDeployment: $agentDeployment) {
		id
	}
}
`

const updateDeploymentMutation = `
mutation($id: String!, $agentDeployment: AgentDeploymentInputUpdate!) {
	updateAgentDeployment(id: $id, agentDeployment: $agentDeployment) {
		id
	}
}
`

func privateMsg(msg string) {
	log.Println(msg)
}

func publicMsg(msg string) {
	log.Println(msg)
	fmt.Println(msg)
}

func main() {

	f, err := os.OpenFile("/var/log/dotgit-hook-postreceive.log", os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	envGitRepoID := os.Getenv("GIT_REPO_ID")
	envGitRepoName := os.Getenv("GIT_REPO_NAME")
	envGitRepoOwner := os.Getenv("GIT_REPO_OWNER")
	envGitRepoPath := os.Getenv("GIT_REPO_PATH")
	envAPIURL := os.Getenv("API_URL")

	log.Println(
		"Starting dotgit-hook-postreceive", os.Args,
		"GIT_REPO_ID="+envGitRepoID,
		"GIT_REPO_NAME="+envGitRepoName,
		"GIT_REPO_OWNER="+envGitRepoOwner,
		"GIT_REPO_PATH="+envGitRepoPath,
		"API_URL="+envAPIURL,
	)

	if envGitRepoID == "" {
		privateMsg("Error: $GIT_REPO_ID is missing")
		os.Exit(1)
	}

	if envGitRepoName == "" {
		privateMsg("Error: $GIT_REPO_NAME is missing")
		os.Exit(1)
	}

	if envGitRepoPath == "" {
		privateMsg("Error: $GIT_REPO_PATH is missing")
		os.Exit(1)
	}

	if envGitRepoOwner == "" {
		privateMsg("Error: $GIT_REPO_OWNER is missing")
		os.Exit(1)
	}

	if envAPIURL == "" {
		privateMsg("Error: $API_URL is missing")
		os.Exit(1)
	}

	// GIT_USER := os.Getenv("GIT_USER")
	// GIT_REPO_OWNER := os.Getenv("GIT_REPO_OWNER")
	// GIT_CLONE_URL := os.Getenv("GIT_CLONE_URL")
	// GIT_OPERATION := os.Getenv("GIT_OPERATION")
	// MQ_HOST := os.Getenv("MQ_HOST")
	// DOCKER_HOST := os.Getenv("DOCKER_HOST")

	gql := graphql.MakeClient(envAPIURL)

	gitbin, err := exec.LookPath("git")
	if err != nil {
		privateMsg("Error: git not found in $PATH")
		os.Exit(1)
	}

	// Fetch last commit SHA1 and message
	cmd := exec.Command(
		gitbin,
		"-C", envGitRepoPath,
		"log", "-1",
		"--pretty=format:%H|%s",
	)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		privateMsg("Error: failed to get informations about latest git commit")
		os.Exit(1)
	}

	// On parse le résultat pour obtenir le SHA1 et le message de commit
	parts := strings.SplitN(string(stdoutStderr), "|", 2)
	if len(parts) < 2 {
		privateMsg("Error: failed to parse informations about latest git commit")
		os.Exit(1)
	}

	newSha1 := parts[0]
	message := parts[1]

	createJSON, err := gql.RequestSync(
		graphql.NewQuery(createDeploymentMutation).SetVariables(graphql.Variables{
			"agentDeployment": graphql.Variables{
				"agentId":       envGitRepoID,
				"pushedAt":      time.Now().Format(time.RFC822Z),
				"commitSHA1":    newSha1,
				"commitMessage": message,
				"buildStatus":   gqltypes.AgentDeployBuildStatus.Pending,
			},
		}),
	)
	if err != nil {
		privateMsg("Error: Could not create pending agent deployment; " + err.Error())
		os.Exit(1)
	}

	var createResponse struct {
		CreateAgentDeployment struct {
			ID string `json:"id"`
		} `json:"createAgentDeployment"`
	}
	json.Unmarshal(createJSON, &createResponse)
	deploymentID := createResponse.CreateAgentDeployment.ID

	updateDeployment := func(deploymentID string, status int, isError bool) error {
		_, err = gql.RequestSync(
			graphql.NewQuery(updateDeploymentMutation).SetVariables(graphql.Variables{
				"id": deploymentID,
				"agentDeployment": graphql.Variables{
					"buildStatus": status,
					"buildError":  isError,
				},
			}),
		)

		return err
	}

	err = updateDeployment(deploymentID, gqltypes.AgentDeployBuildStatus.Building, false)
	if err != nil {
		privateMsg("Error: Could not set agent deployment ID=" + deploymentID + " to 'Building'; " + err.Error())
		os.Exit(1)
	}

	err = build(message, envGitRepoPath, envGitRepoOwner+"/"+envGitRepoName)
	if err != nil {
		privateMsg("Error: could not build agent; " + err.Error())
		updateDeployment(deploymentID, gqltypes.AgentDeployBuildStatus.Finished, true)
		os.Exit(1)
	}

	err = updateDeployment(deploymentID, gqltypes.AgentDeployBuildStatus.Finished, false)
	if err != nil {
		privateMsg("Error: Could not set agent deployment ID=" + deploymentID + " to 'Finished'")
		os.Exit(1)
	}
}

func build(message, repourl, imagename string) error {

	// On lance le build
	builderbin, err := exec.LookPath("agentbuilder-cli")
	if err != nil {
		return errors.New("Error: agentbuilder-cli not found in $PATH")
	}

	cmd2 := exec.Command(
		builderbin,
		"--repourl", repourl,
		"--imagename", imagename,
	)
	cmd2.Env = os.Environ()

	//cmd2.Stdin = os.Stdin
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr

	err = cmd2.Start()
	if err != nil {
		return errors.New("Error: agentbuilder-cli could not be ran")
	}

	err = cmd2.Wait()
	if err != nil {
		return errors.New("Error: failed to build agent; " + err.Error())
	}

	return nil
}
