package arenaserver

import (
	"sync"

	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	"github.com/bytearena/bytearena/arenaserver/container"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	uuid "github.com/satori/go.uuid"
)

const debug = false

type Server struct {
	host                  string
	arenaServerUUID       string
	port                  int
	stopticking           chan bool
	tickspersec           int
	containerorchestrator container.ContainerOrchestrator
	agents                map[uuid.UUID]agent.AgentInterface
	agentsmutex           *sync.Mutex
	state                 *state.ServerState
	commserver            *comm.CommServer
	nbhandshaked          int
	currentturn           utils.Tickturn
	currentturnmutex      *sync.Mutex
	stateobservers        []chan state.ServerState
	debugNbMutations      int
	debugNbUpdates        int
	mqClient              mq.ClientInterface

	agentimages map[uuid.UUID]string

	agenthandshakes map[uuid.UUID]struct{}

	game GameInterface

	tearDownCallbacks      []types.TearDownCallback
	tearDownCallbacksMutex *sync.Mutex

	collisionListener *CollisionListener
}

func NewServer(host string, port int, orch container.ContainerOrchestrator, game GameInterface, arenaServerUUID string, mqClient mq.ClientInterface) *Server {

	gamehost := host

	if host == "" {
		host, err := orch.GetHost(&orch)
		utils.Check(err, "Could not determine arena-server host/ip.")

		gamehost = host
	}

	s := &Server{
		host:                  gamehost,
		arenaServerUUID:       arenaServerUUID,
		port:                  port,
		stopticking:           make(chan bool),
		tickspersec:           game.GetTps(),
		containerorchestrator: orch,
		agents:                make(map[uuid.UUID]agent.AgentInterface),
		agentsmutex:           &sync.Mutex{},
		state:                 state.NewServerState(game.GetMapContainer()),
		commserver:            nil, // initialized in Listen()
		nbhandshaked:          0,
		currentturnmutex:      &sync.Mutex{},
		mqClient:              mqClient,

		agentimages: make(map[uuid.UUID]string),

		agenthandshakes: make(map[uuid.UUID]struct{}),

		game: game,

		tearDownCallbacks:      make([]types.TearDownCallback, 0),
		tearDownCallbacksMutex: &sync.Mutex{},
	}

	s.collisionListener = newCollisionListener(s)
	s.state.PhysicalWorld.SetContactListener(s.collisionListener)

	s.state.PhysicalWorld.SetContactFilter(newCollisionFilter(s))

	return s
}

///////////////////////////////////////////////////////////////////////////////
// Public API
///////////////////////////////////////////////////////////////////////////////

func (s *Server) GetGame() GameInterface {
	return s.game
}

func (s *Server) GetState() *state.ServerState {
	return s.state
}

func (s *Server) GetTicksPerSecond() int {
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
	return len(s.game.GetContestants())
}
