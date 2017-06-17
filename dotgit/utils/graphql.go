package utils

import (
	"strconv"

	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/dotgit/protocol"
)

func GqlUserToUser(gqluser graphqltype.UserType) protocol.User {
	intownerid, _ := strconv.Atoi(gqluser.Id)

	return protocol.User{
		ID:              uint(intownerid),
		Username:        gqluser.Username,
		Name:            gqluser.Name,
		Email:           gqluser.Email,
		UniversalReader: gqluser.UniversalReader,
		UniversalWriter: gqluser.UniversalWriter,
	}
}

func GqlAgentToRepo(gqlagent graphqltype.AgentType) protocol.GitRepository {

	intid, _ := strconv.Atoi(gqlagent.Id)
	owner := GqlUserToUser(gqlagent.Owner)

	return protocol.GitRepository{
		ID:       uint(intid),
		Name:     gqlagent.GitRepository.Name,
		CloneURL: gqlagent.GitRepository.CloneURL,
		Ref:      gqlagent.GitRepository.Ref,
		OwnerID:  int(owner.ID),
		Owner:    owner,
	}
}

func GqlPubKeyToPubKey(gqlpubkey graphqltype.SSHPublicKeyType) protocol.GitPublicKey {
	owner := GqlUserToUser(gqlpubkey.Owner)

	return protocol.GitPublicKey{
		OwnerID:     int(owner.ID),
		Owner:       owner,
		KeyName:     gqlpubkey.Name,
		KeyType:     gqlpubkey.Type,
		Key:         gqlpubkey.Key,
		Fingerprint: gqlpubkey.Fingerprint,
		Comment:     gqlpubkey.Comment,
	}
}
