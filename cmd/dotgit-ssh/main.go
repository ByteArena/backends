package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/bytearena/bytearena/dotgit/config"
	"github.com/bytearena/bytearena/dotgit/database"
	"github.com/bytearena/bytearena/dotgit/protocol"
	"github.com/bytearena/bytearena/dotgit/utils"
)

func main() {

	cnf := config.GetConfig()

	f, err := os.OpenFile("/var/log/dotgit-ssh.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println("Starting a dotgit-ssh session", os.Args, os.Getenv("SSH_ORIGINAL_COMMAND"))

	var db protocol.Database = database.NewGraphQLDatabase()

	err = db.Connect(cnf.GetDatabaseURI())
	if err != nil {
		fmt.Println("Cannot connect to database", err)
		log.Println("Cannot connect to database", err)
		os.Exit(1)
	}

	if len(os.Args) != 2 {
		fmt.Println("Error: fixed git username missing or invalid in call to ", os.Args[0])
		log.Println("Error: fixed git username missing or invalid in call to ", os.Args[0])
		os.Exit(1)
	}

	sshKeyFixedUsername := os.Args[1]
	originalGitCommand := os.Getenv("SSH_ORIGINAL_COMMAND")

	sshKeyFixedUser, err := db.FindUserByUsername(sshKeyFixedUsername)
	if err != nil {
		fmt.Println(err)
		log.Println(err)
		os.Exit(1)
	}

	gitOperation, gitRepoPath, err := parseGitCommand(originalGitCommand)
	if err != nil {
		fmt.Println(err)
		log.Println(err)
		os.Exit(1)
	}

	gitRepoUsername, gitRepoName, err := parseRepositoryName(gitRepoPath)
	if err != nil {
		fmt.Println(err)
		log.Println(err)
		os.Exit(1)
	}

	repoUser, err := db.FindUserByUsername(gitRepoUsername)
	if err != nil {
		fmt.Println(err)
		log.Println(err)
		os.Exit(1)
	}

	repo, err := db.FindRepository(repoUser, gitRepoName)
	if err != nil {
		fmt.Println(err)
		log.Println(err)
		os.Exit(1)
	}

	switch gitOperation {
	case "receive-pack":
		{
			if !hasWritePermission(sshKeyFixedUser, repo) {
				fmt.Println("Write denied to required repository.")
				log.Println("Write denied to required repository.")
				os.Exit(1)
			}
			break
		}
	case "upload-pack":
		{
			if !hasReadPermission(sshKeyFixedUser, repo) {
				fmt.Println("Read denied to required repository.")
				log.Println("Read denied to required repository.")
				os.Exit(1)
			}
			break
		}
	default:
		{
			fmt.Println("Invalid git operation; should be either git-receive-pack or git-upload-pack.")
			log.Println("Invalid git operation; should be either git-receive-pack or git-upload-pack.")
			os.Exit(1)
		}
	}

	err = processGitOperation(repoUser, repo, gitOperation)
	if err != nil {
		fmt.Println(err)
		log.Println(err)
		os.Exit(1)
	}
}

func hasWritePermission(user protocol.User, repo protocol.GitRepository) bool {
	return user.UniversalWriter || uint(repo.OwnerID) == user.ID
}

func hasReadPermission(user protocol.User, repo protocol.GitRepository) bool {
	return user.UniversalReader || uint(repo.OwnerID) == user.ID
}

// Checks whether a command is a valid git command
// The following format is allowed:
// git-([a-z-]+) '/?([\w-+@][\w-+.@]*/)?([\w-]+)\.git'
//
// Taken from github.com/tsuru/gandalf
//
func parseGitCommand(sshOriginalCommand string) (command string, repopath string, err error) {
	r, err := regexp.Compile(`git-([a-z-]+) '/?([\w-+@][\w-+.@]*/)?([\w-]+)\.git'`)
	if err != nil {
		return "", "", errors.New("parseGitCommand(): could not compile regex")
	}

	m := r.FindStringSubmatch(sshOriginalCommand)
	if len(m) != 4 {
		return "", "", errors.New("parseGitCommand(): Invalid GIT command")
	}

	return m[1], m[2] + m[3], nil
}

func parseRepositoryName(repopath string) (username string, reponame string, err error) {
	parts := strings.Split(repopath, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", errors.New("Cannot parse repository name; should be username/reponame")
	}

	return parts[0], parts[1], nil
}

func processGitOperation(user protocol.User, repo protocol.GitRepository, gitOperation string) error {

	cnf := config.GetConfig()

	repoAbsPath := path.Join(cnf.GetGitRepositoriesPath(), utils.RepoRelPath(repo))

	gitbin, err := exec.LookPath("git")
	if err != nil {
		return errors.New("Error: git not found in $PATH")
	}

	cmd := exec.Command(
		gitbin,
		gitOperation,
		repoAbsPath,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	cmd.Env = append(
		os.Environ(),
		"GIT_USER="+user.Username,
		"GIT_REPO_ID="+string(repo.Owner.ID),
		"GIT_REPO_OWNER="+repo.Owner.Username,
		"GIT_REPO_NAME="+repo.Name,
		"GIT_REPO_PATH="+repoAbsPath,
		"GIT_CLONE_URL="+repo.CloneURL,
		"GIT_OPERATION="+gitOperation,
		"MQ_HOST="+cnf.GetMqHost(),
		"DOCKER_HOST="+cnf.GetDockerHost(),
		"API_URL="+cnf.GetDatabaseURI(),
	)

	err = cmd.Run()
	if err != nil {
		return errors.New("Error: error during git operation; " + stderr.String())
	}

	// Git operation successful (+ post-receive hook that built the agent)

	return nil
}
