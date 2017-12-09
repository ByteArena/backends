package container

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/bytearena/core/common/utils"
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
