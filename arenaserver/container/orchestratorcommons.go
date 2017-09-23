package container

import (
	"errors"
	"io"
	"os"
	"strconv"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	uuid "github.com/satori/go.uuid"

	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/utils"
)

func normalizeDockerRef(dockerimage string) (string, error) {

	p, _ := reference.Parse(dockerimage)
	named, ok := p.(reference.Named)
	if !ok {
		return "", errors.New("Invalid docker image name")
	}

	parsedRefWithTag := reference.TagNameOnly(named)
	return parsedRefWithTag.String(), nil
}

func commonCreateAgentContainer(orch arenaservertypes.ContainerOrchestrator, agentid uuid.UUID, host string, port int, dockerimage string) (*arenaservertypes.AgentContainer, error) {
	containerUnixUser := utils.GetenvOrDefault("CONTAINER_UNIX_USER", "root")

	normalizedDockerimage, err := normalizeDockerRef(dockerimage)

	if err != nil {
		return nil, err
	}

	localimages, _ := orch.GetCli().ImageList(orch.GetContext(), types.ImageListOptions{})
	foundlocal := false
	for _, localimage := range localimages {
		for _, alias := range localimage.RepoTags {
			if normalizedAlias, err := normalizeDockerRef(alias); err == nil {
				if normalizedAlias == normalizedDockerimage {
					foundlocal = true
					break
				}
			}
		}

		if foundlocal {
			break
		}
	}

	if !foundlocal {
		reader, err := orch.GetCli().ImagePull(
			orch.GetContext(),
			dockerimage,
			types.ImagePullOptions{
				RegistryAuth: orch.GetRegistryAuth(),
			},
		)

		if err != nil {
			return nil, errors.New("Failed to pull " + dockerimage + " from registry; " + err.Error())
		}

		defer reader.Close()

		io.Copy(os.Stdout, reader)
		utils.Debug("orch", "Pulled image successfully")
	}

	containerconfig := container.Config{
		Image: normalizedDockerimage,
		User:  containerUnixUser,
		Env: []string{
			"PORT=" + strconv.Itoa(port),
			"HOST=" + host,
			"AGENTID=" + agentid.String(),
		},
		AttachStdout: false,
		AttachStderr: false,
	}

	hostconfig := container.HostConfig{
		CapDrop:        []string{"ALL"},
		Privileged:     false,
		AutoRemove:     true,
		ReadonlyRootfs: true,
		NetworkMode:    "bridge",
		// Resources: container.Resources{
		// 	Memory: 1024 * 1024 * 32, // 32M
		// 	//CPUQuota: 5 * (1000),       // 5% en cent-millièmes
		// 	//CPUShares: 1,
		// 	CPUPercent: 5,
		// },
	}

	resp, err := orch.GetCli().ContainerCreate(
		orch.GetContext(), // go context
		&containerconfig,  // container config
		&hostconfig,       // host config
		nil,               // network config
		"agent-"+agentid.String(), // container name
	)
	if err != nil {
		return nil, errors.New("Failed to create docker container for agent " + agentid.String() + "; " + err.Error())
	}

	agentcontainer := arenaservertypes.NewAgentContainer(agentid, arenaservertypes.ContainerId(resp.ID), normalizedDockerimage)
	orch.AddContainer(agentcontainer)

	return agentcontainer, nil
}