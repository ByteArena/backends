package arenaserver

import (
	"sync"

	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	uuid "github.com/satori/go.uuid"

	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/game/deathmatch"
)

const debug = false

type Server struct {
	host            string
	port            int
	arenaServerUUID string
	tickspersec     int

	stopticking      chan bool
	nbhandshaked     int
	currentturn      utils.Tickturn
	currentturnmutex *sync.Mutex
	debugNbMutations int
	debugNbUpdates   int

	stateobservers         []chan interface{}
	tearDownCallbacks      []types.TearDownCallback
	tearDownCallbacksMutex *sync.Mutex

	containerorchestrator arenaservertypes.ContainerOrchestrator

	commserver *comm.CommServer
	mqClient   mq.ClientInterface

	gameDescription        types.GameDescriptionInterface
	agentproxies           map[uuid.UUID]agent.AgentProxyInterface
	agentproxiesmutex      *sync.Mutex
	agentimages            map[uuid.UUID]string
	agentproxieshandshakes map[uuid.UUID]struct{}

	// State

	game *deathmatch.DeathmatchGame

	pendingmutations []protocol.AgentMutationBatch
	mutationsmutex   *sync.Mutex
}

func NewServer(host string, port int, orch arenaservertypes.ContainerOrchestrator, gameDescription types.GameDescriptionInterface, arenaServerUUID string, mqClient mq.ClientInterface) *Server {

	gamehost := host

	if host == "" {
		host, err := orch.GetHost()
		utils.Check(err, "Could not determine arena-server host/ip.")

		gamehost = host
	}

	s := &Server{
		host:            gamehost,
		port:            port,
		arenaServerUUID: arenaServerUUID,
		tickspersec:     gameDescription.GetTps(),

		stopticking:      make(chan bool),
		nbhandshaked:     0,
		currentturnmutex: &sync.Mutex{},

		tearDownCallbacks:      make([]types.TearDownCallback, 0),
		tearDownCallbacksMutex: &sync.Mutex{},

		containerorchestrator: orch,
		commserver:            nil, // initialized in Listen()
		mqClient:              mqClient,

		gameDescription: gameDescription,

		// agents here: proxy to agent in container
		agentproxies:           make(map[uuid.UUID]agent.AgentProxyInterface),
		agentproxiesmutex:      &sync.Mutex{},
		agentproxieshandshakes: make(map[uuid.UUID]struct{}),
		agentimages:            make(map[uuid.UUID]string),

		pendingmutations: make([]protocol.AgentMutationBatch, 0),
		mutationsmutex:   &sync.Mutex{},

		///////////////////////////////////////////////////////////////////////
		// Game logic
		///////////////////////////////////////////////////////////////////////

		game: deathmatch.NewDeathmatchGame(gameDescription),
	}

	return s
}

///////////////////////////////////////////////////////////////////////////////
// Public API
///////////////////////////////////////////////////////////////////////////////

func (s Server) GetGameDescription() types.GameDescriptionInterface {
	return s.gameDescription
}

func (s Server) GetGame() *deathmatch.DeathmatchGame {
	return s.game
}

func (s Server) GetTicksPerSecond() int {
	return s.tickspersec
}

func (s *Server) setTurn(turn utils.Tickturn) {
	s.currentturnmutex.Lock()
	defer s.currentturnmutex.Unlock()

	s.currentturn = turn
}

func (s *Server) getTurn() utils.Tickturn {
	s.currentturnmutex.Lock()
	defer s.currentturnmutex.Unlock()

	res := s.currentturn

	return res
}

func (s *Server) getNbExpectedagents() int {
	return len(s.GetGameDescription().GetContestants())
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
// OLD state
///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

func (s *Server) ProcessMutations() {

	s.mutationsmutex.Lock()
	mutations := s.pendingmutations
	s.pendingmutations = make([]protocol.AgentMutationBatch, 0)
	s.mutationsmutex.Unlock()

	s.game.ProcessMutations(mutations)
}
