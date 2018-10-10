// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package won implements the WorldOpenNetwork protocol.
package won

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/worldopennetwork/go-won/accounts"
	"github.com/worldopennetwork/go-won/common"
	"github.com/worldopennetwork/go-won/common/hexutil"
	"github.com/worldopennetwork/go-won/consensus"
	"github.com/worldopennetwork/go-won/consensus/clique"
	"github.com/worldopennetwork/go-won/consensus/dpos"
	"github.com/worldopennetwork/go-won/consensus/ethash"
	"github.com/worldopennetwork/go-won/core"
	"github.com/worldopennetwork/go-won/core/bloombits"
	"github.com/worldopennetwork/go-won/core/types"
	"github.com/worldopennetwork/go-won/core/vm"
	"github.com/worldopennetwork/go-won/event"
	"github.com/worldopennetwork/go-won/internal/wonapi"
	"github.com/worldopennetwork/go-won/log"
	"github.com/worldopennetwork/go-won/miner"
	"github.com/worldopennetwork/go-won/node"
	"github.com/worldopennetwork/go-won/p2p"
	"github.com/worldopennetwork/go-won/params"
	"github.com/worldopennetwork/go-won/rlp"
	"github.com/worldopennetwork/go-won/rpc"
	"github.com/worldopennetwork/go-won/won/downloader"
	"github.com/worldopennetwork/go-won/won/filters"
	"github.com/worldopennetwork/go-won/won/gasprice"
	"github.com/worldopennetwork/go-won/wondb"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// WorldOpenNetwork implements the WorldOpenNetwork full node service.
type WorldOpenNetwork struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the WorldOpenNetwork
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb wondb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *EthApiBackend

	miner    *miner.Miner
	gasPrice *big.Int
	wonbase  common.Address

	networkId     uint64
	netRPCService *wonapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and wonbase)
}

func (s *WorldOpenNetwork) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new WorldOpenNetwork object (including the
// initialisation of the common WorldOpenNetwork object)
func New(ctx *node.ServiceContext, config *Config) (*WorldOpenNetwork, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run won.WorldOpenNetwork in light sync mode, use les.LightEthereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	won := &WorldOpenNetwork{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.Ethash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		wonbase:        config.Wonbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising WorldOpenNetwork protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gwon upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	won.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, won.chainConfig, won.engine, vmConfig)
	if err != nil {
		return nil, err
	}

	//if chainConfig.Dpos !=nil {
	//	won.engine.APIs(won.blockchain)
	//}

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		won.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	won.bloomIndexer.Start(won.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	won.txPool = core.NewTxPool(config.TxPool, won.chainConfig, won.blockchain)

	if won.protocolManager, err = NewProtocolManager(won.chainConfig, config.SyncMode, config.NetworkId, won.eventMux, won.txPool, won.engine, won.blockchain, chainDb); err != nil {
		return nil, err
	}
	won.miner = miner.New(won, won.chainConfig, won.EventMux(), won.engine)
	won.miner.SetExtra(makeExtraData(config.ExtraData))

	won.ApiBackend = &EthApiBackend{won, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	won.ApiBackend.gpo = gasprice.NewOracle(won.ApiBackend, gpoParams)

	return won, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gwon",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (wondb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*wondb.LDBDatabase); ok {
		db.Meter("won/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an WorldOpenNetwork service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ethash.Config, chainConfig *params.ChainConfig, db wondb.Database) consensus.Engine {

	//if dpos is request
	if chainConfig.Dpos != nil {
		return dpos.New(chainConfig.Dpos, db)
	}

	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowMode == ethash.ModeFake:
		log.Warn("Ethash used in fake mode")
		return ethash.NewFaker()
	case config.PowMode == ethash.ModeTest:
		log.Warn("Ethash used in test mode")
		return ethash.NewTester()
	case config.PowMode == ethash.ModeShared:
		log.Warn("Ethash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ethash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *WorldOpenNetwork) APIs() []rpc.API {
	apis := wonapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "won",
			Version:   "1.0",
			Service:   NewPublicWorldOpenNetworkAPI(s),
			Public:    true,
		}, {
			Namespace: "won",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "won",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "won",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *WorldOpenNetwork) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *WorldOpenNetwork) Wonbase() (eb common.Address, err error) {
	s.lock.RLock()
	wonbase := s.wonbase
	s.lock.RUnlock()

	if wonbase != (common.Address{}) {
		return wonbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			wonbase := accounts[0].Address

			s.lock.Lock()
			s.wonbase = wonbase
			s.lock.Unlock()

			log.Info("Wonbase automatically configured", "address", wonbase)
			return wonbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("wonbase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *WorldOpenNetwork) SetWonbase(wonbase common.Address) {
	self.lock.Lock()
	self.wonbase = wonbase
	self.lock.Unlock()

	self.miner.SetWonbase(wonbase)
}

func (s *WorldOpenNetwork) StartMining(local bool) error {
	eb, err := s.Wonbase()
	if err != nil {
		log.Error("Cannot start mining without wonbase", "err", err)
		return fmt.Errorf("wonbase missing: %v", err)
	}

	if dpos, ok := s.engine.(*dpos.Dpos); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Wonbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		dpos.Authorize(eb, wallet.SignHash)
	}

	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Wonbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so none will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *WorldOpenNetwork) StopMining()         { s.miner.Stop() }
func (s *WorldOpenNetwork) IsMining() bool      { return s.miner.Mining() }
func (s *WorldOpenNetwork) Miner() *miner.Miner { return s.miner }

func (s *WorldOpenNetwork) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *WorldOpenNetwork) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *WorldOpenNetwork) TxPool() *core.TxPool               { return s.txPool }
func (s *WorldOpenNetwork) EventMux() *event.TypeMux           { return s.eventMux }
func (s *WorldOpenNetwork) Engine() consensus.Engine           { return s.engine }
func (s *WorldOpenNetwork) ChainDb() wondb.Database            { return s.chainDb }
func (s *WorldOpenNetwork) IsListening() bool                  { return true } // Always listening
func (s *WorldOpenNetwork) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *WorldOpenNetwork) NetVersion() uint64                 { return s.networkId }
func (s *WorldOpenNetwork) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *WorldOpenNetwork) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// WorldOpenNetwork protocol implementation.
func (s *WorldOpenNetwork) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = wonapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// WorldOpenNetwork protocol.
func (s *WorldOpenNetwork) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
