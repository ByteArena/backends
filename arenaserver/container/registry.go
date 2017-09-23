package container

import (
	"context"

	"github.com/bytearena/bytearena/common/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/client"
)

func registryLogin(address string, ctx context.Context, client *client.Client) string {
	auth := types.AuthConfig{
		ServerAddress: address,
	}

	_, err := client.RegistryLogin(ctx, auth)
	utils.Check(err, "Failed to log onto docker registry")

	// FIXME(sven): there is no auth yet
	// return res.IdentityToken
	return "no_token"
}

func (orch *ContainerOrchestrator) publishInRegistry(image string) {
	options := types.ImagePushOptions{
		All:          true,
		RegistryAuth: orch.registryAuth,
	}

	_, err := orch.cli.ImagePush(
		orch.ctx,
		image,
		options,
	)
	utils.Check(err, "Failed to push docker image to registry")

	// TODO(sven): fix reader to avoid ressource leakage
	// readCloser.Closer.Close()
}
