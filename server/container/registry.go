package container

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func registryLogin(ctx context.Context, client *client.Client) string {
	auth := types.AuthConfig{
		ServerAddress: "127.0.0.1:5000",
	}

	_, err := client.RegistryLogin(ctx, auth)

	if err != nil {
		log.Panicln("Error during login:", err)
	}

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

	if err != nil {
		log.Panicln("Error during image push:", err)
	}

	// TODO(sven): fix reader to avoid ressource leakage
	// readCloser.Closer.Close()
}
