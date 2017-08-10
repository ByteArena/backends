package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytearena/bytearena/dotgit/config"
	"github.com/bytearena/bytearena/dotgit/database"
	"github.com/bytearena/bytearena/dotgit/protocol"
	"github.com/bytearena/bytearena/dotgit/utils"
)

func msgOut(msg string) {
	log.Println(msg)
	os.Exit(1)
}

func errorCheck(err error, msg string) {
	if err == nil {
		return
	}

	msgOut(msg + "; " + err.Error()) // logfile
}

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
	errorCheck(err, "Cannot connect to database")

	if len(os.Args) != 2 {
		msgOut("Error: fixed git username missing or invalid")
	}

	sshKeyFixedUsername := os.Args[1]
	originalGitCommand := os.Getenv("SSH_ORIGINAL_COMMAND")

	sshKeyFixedUser, err := db.FindUserByUsername(sshKeyFixedUsername)
	errorCheck(err, "Error: cannot find user")

	gitOperation, gitRepoPath, err := parseGitCommand(originalGitCommand)
	errorCheck(err, "Error: cannot parse git command")

	gitRepoUsername, gitRepoName, err := parseRepositoryName(gitRepoPath)
	errorCheck(err, "Error: cannot parse repository name")

	repoUser, err := db.FindUserByUsername(gitRepoUsername)
	errorCheck(err, "Error: cannot determine username associated to repository")

	repo, err := db.FindRepository(repoUser, gitRepoName)
	errorCheck(err, "Error: cannot find corresponding repository")

	switch gitOperation {
	case "receive-pack":
		{
			if !hasWritePermission(sshKeyFixedUser, repo) {
				msgOut("Error: Write denied to required repository.")
			}
			break
		}
	case "upload-pack":
		{
			if !hasReadPermission(sshKeyFixedUser, repo) {
				msgOut("Error: Read denied to required repository.")
			}
			break
		}
	default:
		{
			msgOut("Error: Invalid git operation; should be either git-receive-pack or git-upload-pack.")
		}
	}

	err = processGitOperation(repoUser, repo, gitOperation)
	errorCheck(err, "Error: Cannot process git operation.")
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
		"GIT_REPO_ID="+strconv.Itoa(int(repo.ID)),
		"GIT_REPO_OWNER="+repo.Owner.Username,
		"GIT_REPO_NAME="+repo.Name,
		"GIT_REPO_PATH="+repoAbsPath,
		"GIT_CLONE_URL="+repo.CloneURL,
		"GIT_OPERATION="+gitOperation,
		"MQ_HOST="+cnf.GetMqHost(),
		"API_URL="+cnf.GetDatabaseURI(),

		"DOCKER_HOST="+cnf.GetDockerHost(),
		"DOCKER_BUILD_MEMORY_LIMIT="+cnf.DockerBuildMemoryLimit,
		"DOCKER_BUILD_NETWORK="+cnf.DockerBuildNetwork,
		"DOCKER_BUILD_NO_CACHE"+cnf.DockerBuildNoCache,
		"DOCKER_BUILD_CPU_PERIOD"+cnf.DockerBuildCpuPeriod,
	)

	err = cmd.Run()
	if err != nil {
		return errors.New("Error: error during git operation; " + stderr.String())
	}

	// Git operation successful (+ post-receive hook that built the agent)

	return nil
}
