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
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/5sWind/bgmchain/accounts"
	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/common/hexutil"
	"github.com/5sWind/bgmchain/consensus"
	"github.com/5sWind/bgmchain/consensus/dpos"
	"github.com/5sWind/bgmchain/core"
	"github.com/5sWind/bgmchain/core/bloombits"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/bgm"
	"github.com/5sWind/bgmchain/bgm/downloader"
	"github.com/5sWind/bgmchain/bgm/filters"
	"github.com/5sWind/bgmchain/bgm/gasprice"
	"github.com/5sWind/bgmchain/bgmdb"
	"github.com/5sWind/bgmchain/event"
	"github.com/5sWind/bgmchain/internal/bgmapi"
	"github.com/5sWind/bgmchain/light"
	"github.com/5sWind/bgmchain/log"
	"github.com/5sWind/bgmchain/node"
	"github.com/5sWind/bgmchain/p2p"
	"github.com/5sWind/bgmchain/p2p/discv5"
	"github.com/5sWind/bgmchain/params"
	rpc "github.com/5sWind/bgmchain/rpc"
)

type LightBgmchain struct {
	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
//
	shutdownChan chan bool
//
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
//
	chainDb bgmdb.Database //

	bloomRequests                              chan chan *bloombits.Retrieval //
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *bgmapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *bgm.Config) (*LightBgmchain, error) {
	chainDb, err := bgm.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lbgm := &LightBgmchain{
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           dpos.New(chainConfig.Dpos, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     bgm.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	lbgm.relay = NewLesTxRelay(peers, lbgm.reqDist)
	lbgm.serverPool = newServerPool(chainDb, quitSync, &lbgm.wg)
	lbgm.retriever = newRetrieveManager(peers, lbgm.reqDist, lbgm.serverPool)
	lbgm.odr = NewLesOdr(chainDb, lbgm.chtIndexer, lbgm.bloomTrieIndexer, lbgm.bloomIndexer, lbgm.retriever)
	if lbgm.blockchain, err = light.NewLightChain(lbgm.odr, lbgm.chainConfig, lbgm.engine); err != nil {
		return nil, err
	}
	lbgm.bloomIndexer.Start(lbgm.blockchain)
//
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lbgm.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lbgm.txPool = light.NewTxPool(lbgm.chainConfig, lbgm.blockchain, lbgm.relay)
	if lbgm.protocolManager, err = NewProtocolManager(lbgm.chainConfig, true, ClientProtocolVersions, config.NetworkId, lbgm.eventMux, lbgm.engine, lbgm.peers, lbgm.blockchain, nil, chainDb, lbgm.odr, lbgm.relay, quitSync, &lbgm.wg); err != nil {
		return nil, err
	}
	lbgm.ApiBackend = &LesApiBackend{lbgm, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	lbgm.ApiBackend.gpo = gasprice.NewOracle(lbgm.ApiBackend, gpoParams)
	return lbgm, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

//
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

//
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

//
func (s *LightDummyAPI) Mining() bool {
	return false
}

//
//
func (s *LightBgmchain) APIs() []rpc.API {
	return append(bgmapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "bgm",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "bgm",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightBgmchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightBgmchain) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightBgmchain) TxPool() *light.TxPool              { return s.txPool }
func (s *LightBgmchain) Engine() consensus.Engine           { return s.engine }
func (s *LightBgmchain) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightBgmchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightBgmchain) EventMux() *event.TypeMux           { return s.eventMux }

//
//
func (s *LightBgmchain) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

//
//
func (s *LightBgmchain) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = bgmapi.NewPublicNetAPI(srvr, s.networkId)
//
//
	protocolVersion := ClientProtocolVersions[len(ClientProtocolVersions)-1]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start()
	return nil
}

//
//
func (s *LightBgmchain) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
