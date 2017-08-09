package database

// test command : go build && DOTGIT_CONFIG=../../dev.conf SSH_ORIGINAL_COMMAND="git-upload-pack 'netgusto/repo-name.git'" ./dotgit-ssh netgusto

import (
	"encoding/json"
	"errors"

	"github.com/bytearena/bytearena/common/graphql"
	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/dotgit/protocol"
	"github.com/bytearena/bytearena/dotgit/utils"
)

const fragmentUser = `
fragment userFields on User {
	id
	name
	username
	email
	universalReader
	universalWriter
}
`

const fetchUserQuery = fragmentUser + `
query ($username: String) {
	users(username: $username) {
		...userFields
	}
}
`

const fetchRepoQuery = fragmentUser + `
query ($username: String, $reponame: String, $id: String) {
	agents(username: $username, name: $reponame, id: $id) {
		id
		name
		gitRepository {
			cloneURL
			username
			name
			ref
		}
		image {
			name
			tag
			registry
		}
		owner {
			...userFields
		}
	}
}
`

const fetchSSHPublicKeyQuery = fragmentUser + `
query ($username: String, $fingerprint: String) {
	sshpublickeys(username: $username, fingerprint: $fingerprint) {
		owner {
			...userFields
		}
		name
		type
		key
		fingerprint
		comment
	}
}
`

const createSSHPublicKeyMutation = fragmentUser + `
mutation ($key: SSHPublicKeyInputCreate!) {
	createSSHPublicKey(key: $key) {
		owner {
			...userFields
		}
		name
		type
		key
		fingerprint
		comment
	}
}
`

type GraphqlDatabase struct {
	client *graphql.Client
}

func NewGraphQLDatabase() *GraphqlDatabase {
	return &GraphqlDatabase{}
}

func (db *GraphqlDatabase) Connect(connURI string) error {
	db.client = graphql.NewClient(connURI)
	return nil
}

func (db *GraphqlDatabase) ActivateDebug() {}

func (db *GraphqlDatabase) Close() {}

func (db *GraphqlDatabase) Migrate() {}

func (db *GraphqlDatabase) CreateTables() {}

func (db *GraphqlDatabase) findUser(variables graphql.Variables) (protocol.User, error) {
	data, err := db.client.RequestSync(
		graphql.NewQuery(fetchUserQuery).SetVariables(variables),
	)

	if err != nil {
		return protocol.User{}, errors.New("There was an error fetching user")
	}

	var apiresponse struct {
		Users []graphqltype.UserType `json:"users"`
	}
	err = json.Unmarshal(data, &apiresponse)
	if err != nil || len(apiresponse.Users) > 1 {
		return protocol.User{}, errors.New("There was an error fetching user")
	}

	if len(apiresponse.Users) == 0 {
		return protocol.User{}, errors.New("User not found")
	}

	apiuser := apiresponse.Users[0]

	return utils.GqlUserToUser(apiuser), nil
}

func (db *GraphqlDatabase) FindUserByUsername(username string) (protocol.User, error) {
	return db.findUser(graphql.Variables{"username": username})
}

func (db *GraphqlDatabase) FindUserByEmail(email string) (protocol.User, error) {
	return db.findUser(graphql.Variables{"email": email})
}

func (db *GraphqlDatabase) FindRepository(user protocol.User, reponame string) (protocol.GitRepository, error) {

	data, err := db.client.RequestSync(
		graphql.NewQuery(fetchRepoQuery).SetVariables(graphql.Variables{
			"username": user.Username,
			"reponame": reponame,
		}),
	)

	if err != nil {
		return protocol.GitRepository{}, errors.New("There was an error fetching repository; " + err.Error())
	}

	return processFoundRepository(data)
}

func (db *GraphqlDatabase) FindRepositoryById(id string) (protocol.GitRepository, error) {
	data, err := db.client.RequestSync(
		graphql.NewQuery(fetchRepoQuery).SetVariables(graphql.Variables{
			"id": id,
		}),
	)

	if err != nil {
		return protocol.GitRepository{}, errors.New("There was an error fetching repository; " + err.Error())
	}

	return processFoundRepository(data)
}

func processFoundRepository(data json.RawMessage) (protocol.GitRepository, error) {
	var apiresponse struct {
		Agents []graphqltype.AgentType `json:"agents"`
	}
	err := json.Unmarshal(data, &apiresponse)
	if err != nil || len(apiresponse.Agents) > 1 {
		return protocol.GitRepository{}, errors.New("There was an error fetching agent; " + err.Error())
	}

	if len(apiresponse.Agents) == 0 {
		return protocol.GitRepository{}, errors.New("Agent not found")
	}

	apiagent := apiresponse.Agents[0]
	return utils.GqlAgentToRepo(apiagent), nil
}

func (db *GraphqlDatabase) FindPublicKeyByFingerprint(fingerprint string) (protocol.GitPublicKey, error) {

	data, err := db.client.RequestSync(
		graphql.NewQuery(fetchSSHPublicKeyQuery).SetVariables(graphql.Variables{
			"fingerprint": fingerprint,
		}),
	)

	if err != nil {
		return protocol.GitPublicKey{}, errors.New("There was an error fetching public key")
	}

	var apiresponse struct {
		SSHPublicKeys []graphqltype.SSHPublicKeyType `json:"sshpublickeys"`
	}
	err = json.Unmarshal(data, &apiresponse)
	if err != nil || len(apiresponse.SSHPublicKeys) > 1 {
		return protocol.GitPublicKey{}, errors.New("There was an error fetching public key")
	}

	if len(apiresponse.SSHPublicKeys) == 0 {
		return protocol.GitPublicKey{}, errors.New("Public key not found")
	}

	apipubkey := apiresponse.SSHPublicKeys[0]
	return utils.GqlPubKeyToPubKey(apipubkey), nil
}

func (db *GraphqlDatabase) CreateUser(user protocol.User) error {
	// Users are created in web interface
	return errors.New("GraphQL adapter does not implement CreateUser().")
}

func (db *GraphqlDatabase) CreateRepository(repo protocol.GitRepository) error {
	//db.conn.Create(&repo)
	return nil
}

func (db *GraphqlDatabase) CreatePublicKey(key protocol.GitPublicKey) error {
	_, err := db.client.RequestSync(
		graphql.NewQuery(createSSHPublicKeyMutation).SetVariables(graphql.Variables{
			"key": graphql.Variables{
				"ownerid":     key.Owner.ID,
				"name":        key.KeyName,
				"type":        key.KeyType,
				"key":         key.Key,
				"fingerprint": key.Fingerprint,
				"comment":     key.Comment,
			},
		}),
	)

	return err
}

func (db *GraphqlDatabase) DeleteRepository(repo protocol.GitRepository) error {
	return nil
}

func (db *GraphqlDatabase) InjectFixtures(agentBuilderPublicKey string) {}
