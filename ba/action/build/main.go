package build

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/bytearena/bytearena/agentbuilder"
	"github.com/bytearena/bytearena/common/utils"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	bettererrors "github.com/xtuc/better-errors"
)

const (
	DOCKER_BUILD_FILE = "Dockerfile"
)

func welcomeBanner() {
	fmt.Println("=== ")
	fmt.Println("=== 🤖  Welcome! I'm the Byte Arena Builder Bot (the local one)")
	fmt.Println("=== ")
	fmt.Println("")
}

func successBanner(name string) {
	fmt.Println("")
	fmt.Println("=== ")
	fmt.Println("=== ✅  Your agent has been built. Let'em know who's the best!")
	fmt.Println("===    Its name is: " + name)
	fmt.Println("=== ")
	fmt.Println("")
}

func Main(dir string) (bool, error) {

	if dir == "" {
		return true, bettererrors.New("No target directory was specified")
	}

	if is, err := isDirectory(dir); !is {
		return true, err
	}

	if has, err := hasDockerBuildFile(dir); !has {
		return true, err
	}

	cli, err := client.NewEnvClient()

	if err != nil {
		return false, bettererrors.
			New("Failed to initialize Docker").
			With(err)
	}

	welcomeBanner()

	fmt.Println("=== Building your agent now.")
	fmt.Println("")

	name := path.Base(dir)
	err = runDockerBuild(cli, name, dir)

	if err != nil {
		return false, err
	}

	successBanner(name)

	return false, nil
}

func isDirectory(directory string) (bool, error) {

	if _, err := os.Stat(directory); os.IsNotExist(err) {

		return false, bettererrors.
			New("Directory does not exists").
			SetContext("directory", directory)
	} else {

		return true, nil
	}
}

func hasDockerBuildFile(inDirectory string) (bool, error) {

	if _, err := os.Stat(path.Join(inDirectory, DOCKER_BUILD_FILE)); os.IsNotExist(err) {

		return false, bettererrors.
			New("Docker build not found").
			SetContext("in directory", inDirectory).
			SetContext("file", DOCKER_BUILD_FILE)
	} else {

		return true, nil
	}
}

// Build a dir
// The dockerfile must be in the dir
func createTar(dir string) (io.Reader, error) {
	buff := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buff)

	// Assert dockerfile
	dockerfileContents, err := ioutil.ReadFile(path.Join(dir, DOCKER_BUILD_FILE))

	if err != nil {
		return buff, err
	}

	err = failForbiddenInstructions(dockerfileContents)

	if err != nil {
		return buff, err
	}

	err = doTar(tw, dir)

	if err != nil {
		return buff, err
	}

	return buff, nil
}

func removePrefix(str string) string {
	if strings.Contains(str, "/") {

		parts := strings.Split(str, "/")
		return path.Join(parts[1:]...)
	}

	return str
}

func doTar(tw *tar.Writer, dir string) error {
	files, err := ioutil.ReadDir(dir)

	if err != nil {
		return err
	}

	for _, f := range files {
		fqn := path.Join(dir, f.Name())
		relpath := removePrefix(fqn)

		if f.IsDir() {
			err := doTar(tw, fqn)

			if err != nil {
				return err
			}
		} else {
			tw.WriteHeader(&tar.Header{
				Name: relpath,
				Size: f.Size(),
				Mode: int64(f.Mode()),
			})

			contents, err := ioutil.ReadFile(fqn)
			_, err = tw.Write(contents)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func runDockerBuild(cli *client.Client, name, dir string) error {
	ctx := context.Background()

	opts := dockertypes.ImageBuildOptions{
		Tags: []string{name},
	}

	tar, tarErr := createTar(dir)

	if tarErr != nil {
		return tarErr
	}

	resp, err := cli.ImageBuild(ctx, tar, opts)

	if err != nil {
		return bettererrors.
			New("Docker build failed").
			With(err)
	}

	reader := resp.Body

	fd, isTerminal := term.GetFdInfo(os.Stdout)

	if err := jsonmessage.DisplayJSONMessagesStream(reader, os.Stdout, fd, isTerminal, nil); err != nil {
		return err
	}

	reader.Close()

	return nil
}

func failForbiddenInstructions(content []byte) error {
	forbiddenInstructions, err := agentbuilder.DockerfileFindForbiddenInstructions(bytes.NewReader(content))

	if err != nil {
		return err
	}

	for name, _ := range forbiddenInstructions {
		berror := bettererrors.
			New("Forbidden instruction. Remember to remove it when you will to deploy your agent.").
			SetContext("name", name.String())

		utils.FailWith(berror)
	}

	return nil
}
