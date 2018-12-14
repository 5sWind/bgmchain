//
//
//
//
//
//
//
//
//
//
//
//
//
//
//

//
package bgm

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/5sWind/bgmchain/accounts"
	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/common/hexutil"
	"github.com/5sWind/bgmchain/consensus"
	"github.com/5sWind/bgmchain/consensus/dpos"
	"github.com/5sWind/bgmchain/core"
	"github.com/5sWind/bgmchain/core/bloombits"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/core/vm"
	"github.com/5sWind/bgmchain/bgm/downloader"
	"github.com/5sWind/bgmchain/bgm/filters"
	"github.com/5sWind/bgmchain/bgm/gasprice"
	"github.com/5sWind/bgmchain/bgmdb"
	"github.com/5sWind/bgmchain/event"
	"github.com/5sWind/bgmchain/internal/bgmapi"
	"github.com/5sWind/bgmchain/log"
	"github.com/5sWind/bgmchain/miner"
	"github.com/5sWind/bgmchain/node"
	"github.com/5sWind/bgmchain/p2p"
	"github.com/5sWind/bgmchain/params"
	"github.com/5sWind/bgmchain/rlp"
	"github.com/5sWind/bgmchain/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

//
type Bgmchain struct {
	config      *Config
	chainConfig *params.ChainConfig

//
	shutdownChan  chan bool    //
	stopDbUpgrade func() error //

//
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

//
	chainDb bgmdb.Database //

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval //
	bloomIndexer  *core.ChainIndexer             //

	ApiBackend *BgmApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	validator common.Address
	coinbase  common.Address

	networkId     uint64
	netRPCService *bgmapi.PublicNetAPI

	lock sync.RWMutex //
}

func (s *Bgmchain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

//
//
func New(ctx *node.ServiceContext, config *Config) (*Bgmchain, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run bgm.Bgmchain in light sync mode, use les.LightBgmchain")
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

	bgm := &Bgmchain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         dpos.New(chainConfig.Dpos, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		validator:      config.Validator,
		coinbase:       config.Coinbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising Bgmchain protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gbgm upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}
	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	bgm.blockchain, err = core.NewBlockChain(chainDb, bgm.chainConfig, bgm.engine, vmConfig)
	if err != nil {
		return nil, err
	}
//
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		bgm.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	bgm.bloomIndexer.Start(bgm.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	bgm.txPool = core.NewTxPool(config.TxPool, bgm.chainConfig, bgm.blockchain)

	if bgm.protocolManager, err = NewProtocolManager(bgm.chainConfig, config.SyncMode, config.NetworkId, bgm.eventMux, bgm.txPool, bgm.engine, bgm.blockchain, chainDb); err != nil {
		return nil, err
	}
	bgm.miner = miner.New(bgm, bgm.chainConfig, bgm.EventMux(), bgm.engine)
	bgm.miner.SetExtra(makeExtraData(config.ExtraData))

	bgm.ApiBackend = &BgmApiBackend{bgm, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	bgm.ApiBackend.gpo = gasprice.NewOracle(bgm.ApiBackend, gpoParams)

	return bgm, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
//
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gbgm",
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

//
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (bgmdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*bgmdb.LDBDatabase); ok {
		db.Meter("bgm/db/chaindata/")
	}
	return db, nil
}

//
//
func (s *Bgmchain) APIs() []rpc.API {
	apis := bgmapi.GetAPIs(s.ApiBackend)

//
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

//
	return append(apis, []rpc.API{
		{
			Namespace: "bgm",
			Version:   "1.0",
			Service:   NewPublicBgmchainAPI(s),
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "bgm",
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

func (s *Bgmchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Bgmchain) Validator() (validator common.Address, err error) {
	s.lock.RLock()
	validator = s.validator
	s.lock.RUnlock()

	if validator != (common.Address{}) {
		return validator, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return common.Address{}, fmt.Errorf("validator address must be explicitly specified")
}

//
func (self *Bgmchain) SetValidator(validator common.Address) {
	self.lock.Lock()
	self.validator = validator
	self.lock.Unlock()
}

func (s *Bgmchain) Coinbase() (eb common.Address, err error) {
	s.lock.RLock()
	coinbase := s.coinbase
	s.lock.RUnlock()

	if coinbase != (common.Address{}) {
		return coinbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return common.Address{}, fmt.Errorf("coinbase address must be explicitly specified")
}

//
func (self *Bgmchain) SetCoinbase(coinbase common.Address) {
	self.lock.Lock()
	self.coinbase = coinbase
	self.lock.Unlock()

	self.miner.SetCoinbase(coinbase)
}

func (s *Bgmchain) StartMining(local bool) error {
	validator, err := s.Validator()
	if err != nil {
		log.Error("Cannot start mining without validator", "err", err)
		return fmt.Errorf("validator missing: %v", err)
	}
	cb, err := s.Coinbase()
	if err != nil {
		log.Error("Cannot start mining without coinbase", "err", err)
		return fmt.Errorf("coinbase missing: %v", err)
	}

	if dpos, ok := s.engine.(*dpos.Dpos); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: validator})
		if wallet == nil || err != nil {
			log.Error("Coinbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		dpos.Authorize(validator, wallet.SignHash)
	}
	if local {
//
//
//
//
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(cb)
	return nil
}

func (s *Bgmchain) StopMining()         { s.miner.Stop() }
func (s *Bgmchain) IsMining() bool      { return s.miner.Mining() }
func (s *Bgmchain) Miner() *miner.Miner { return s.miner }

func (s *Bgmchain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Bgmchain) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Bgmchain) TxPool() *core.TxPool               { return s.txPool }
func (s *Bgmchain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Bgmchain) Engine() consensus.Engine           { return s.engine }
func (s *Bgmchain) ChainDb() bgmdb.Database            { return s.chainDb }
func (s *Bgmchain) IsListening() bool                  { return true } //
func (s *Bgmchain) BgmVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Bgmchain) NetVersion() uint64                 { return s.networkId }
func (s *Bgmchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

//
//
func (s *Bgmchain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

//
//
func (s *Bgmchain) Start(srvr *p2p.Server) error {
//
	s.startBloomHandlers()

//
	s.netRPCService = bgmapi.NewPublicNetAPI(srvr, s.NetVersion())

//
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		maxPeers -= s.config.LightPeers
		if maxPeers < srvr.MaxPeers/2 {
			maxPeers = srvr.MaxPeers / 2
		}
	}
//
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

//
//
func (s *Bgmchain) Stop() error {
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
