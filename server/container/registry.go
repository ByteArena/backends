package container

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func registryLogin(ctx context.Context, client *client.Client) string {
	auth := types.AuthConfig{}

	res, err := client.RegistryLogin(ctx, auth)

	if err != nil {
		log.Panicln(err)
	}

	return res.IdentityToken
}

func (orch *ContainerOrchestrator) publishInRegistry(image string) {
	options := types.ImagePushOptions{
		RegistryAuth: orch.registryAuth,
	}

	_, err := orch.cli.ImagePush(
		orch.ctx,
		image,
		options,
	)

	if err != nil {
		log.Panicln(err)
	}
}
