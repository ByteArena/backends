package database

// test command : go build && DOTGIT_CONFIG=../../dev.conf SSH_ORIGINAL_COMMAND="git-upload-pack 'netgusto/repo-name.git'" ./dotgit-ssh netgusto

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/bytearena/bytearena/common/graphql"
	graphqltype "github.com/bytearena/bytearena/common/graphql/types"
	"github.com/bytearena/bytearena/dotgit/protocol"
)

const fetchUserQuery = `
	query($username: String) {
		users(username: $username) {
			id
			name
			username
			email
			universalreader
			universalwriter
		}
	}
`

const fetchRepoQuery = `
	query($username: String!, $reponame: String) {
		agents(username: $username, name: $reponame) {
			id
			name
			cloneurl
			image {
				name
				tag
				registry
			}
			owner {
				id
				username
			}
		}
	}
`

const fetchSSHPublicKeyQuery = `
	query($username: String, $fingerprint: String) {
		sshpublickeys(username: $username, fingerprint: $fingerprint) {
			owner {
				id
				name
				username
				email
			}
			name
			type
			key
			fingerprint
			comment
		}
	}
`

const createSSHPublicKeyMutation = `
	mutation($key: SSHPublicKeyInputCreate!) {
		createSSHPublicKey(key: $key) {
			owner {
				id
				name
				username
				email
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

	intid, err := strconv.Atoi(apiuser.Id)

	return protocol.User{
		ID:              uint(intid),
		Username:        apiuser.Username,
		Name:            apiuser.Name,
		Email:           apiuser.Email,
		UniversalReader: apiuser.UniversalReader,
		UniversalWriter: apiuser.UniversalWriter,
	}, nil
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
		return protocol.GitRepository{}, errors.New("There was an error fetching agent")
	}

	var apiresponse struct {
		Agents []graphqltype.AgentType `json:"agents"`
	}
	err = json.Unmarshal(data, &apiresponse)
	if err != nil || len(apiresponse.Agents) > 1 {
		return protocol.GitRepository{}, errors.New("There was an error fetching agent")
	}

	if len(apiresponse.Agents) == 0 {
		return protocol.GitRepository{}, errors.New("Agent not found")
	}

	apiagent := apiresponse.Agents[0]
	intid, err := strconv.Atoi(apiagent.Id)
	intownerid, err := strconv.Atoi(apiagent.Owner.Id)

	return protocol.GitRepository{
		ID:       uint(intid),
		RepoName: apiagent.Image.Name + ":" + apiagent.Image.Tag,
		Title:    apiagent.Name,
		OwnerID:  intownerid,
	}, nil
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
	intownerid, err := strconv.Atoi(apipubkey.Owner.Id)

	return protocol.GitPublicKey{
		OwnerID:     intownerid,
		KeyName:     apipubkey.Name,
		KeyType:     apipubkey.Type,
		Key:         apipubkey.Key,
		Fingerprint: apipubkey.Fingerprint,
		Comment:     apipubkey.Comment,
	}, nil
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
