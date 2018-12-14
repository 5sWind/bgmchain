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

package swarm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"

	"github.com/5sWind/bgmchain/accounts/abi/bind"
	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/contracts/chequebook"
	"github.com/5sWind/bgmchain/contracts/ens"
	"github.com/5sWind/bgmchain/crypto"
	"github.com/5sWind/bgmchain/bgmclient"
	"github.com/5sWind/bgmchain/log"
	"github.com/5sWind/bgmchain/node"
	"github.com/5sWind/bgmchain/p2p"
	"github.com/5sWind/bgmchain/p2p/discover"
	"github.com/5sWind/bgmchain/rpc"
	"github.com/5sWind/bgmchain/swarm/api"
	httpapi "github.com/5sWind/bgmchain/swarm/api/http"
	"github.com/5sWind/bgmchain/swarm/fuse"
	"github.com/5sWind/bgmchain/swarm/network"
	"github.com/5sWind/bgmchain/swarm/storage"
)

//
type Swarm struct {
	config      *api.Config            //
	api         *api.Api               //
	dns         api.Resolver           //
	dbAccess    *network.DbAccess      //
	storage     storage.ChunkStore     //
	dpa         *storage.DPA           //
	depo        network.StorageHandler //
	cloud       storage.CloudStore     //
	hive        *network.Hive          //
	backend     chequebook.Backend     //
	privateKey  *ecdsa.PrivateKey
	corsString  string
	swapEnabled bool
	lstore      *storage.LocalStore //
	sfs         *fuse.SwarmFS       //
}

type SwarmAPI struct {
	Api     *api.Api
	Backend chequebook.Backend
	PrvKey  *ecdsa.PrivateKey
}

func (self *Swarm) API() *SwarmAPI {
	return &SwarmAPI{
		Api:     self.api,
		Backend: self.backend,
		PrvKey:  self.privateKey,
	}
}

//
//
func NewSwarm(ctx *node.ServiceContext, backend chequebook.Backend, ensClient *bgmclient.Client, config *api.Config, swapEnabled, syncEnabled bool, cors string) (self *Swarm, err error) {
	if bytes.Equal(common.FromHex(config.PublicKey), storage.ZeroKey) {
		return nil, fmt.Errorf("empty public key")
	}
	if bytes.Equal(common.FromHex(config.BzzKey), storage.ZeroKey) {
		return nil, fmt.Errorf("empty bzz key")
	}

	self = &Swarm{
		config:      config,
		swapEnabled: swapEnabled,
		backend:     backend,
		privateKey:  config.Swap.PrivateKey(),
		corsString:  cors,
	}
	log.Debug(fmt.Sprintf("Setting up Swarm service components"))

	hash := storage.MakeHashFunc(config.ChunkerParams.Hash)
	self.lstore, err = storage.NewLocalStore(hash, config.StoreParams)
	if err != nil {
		return
	}

//
	log.Debug(fmt.Sprintf("Set up local storage"))

	self.dbAccess = network.NewDbAccess(self.lstore)
	log.Debug(fmt.Sprintf("Set up local db access (iterator/counter)"))

//
	self.hive = network.NewHive(
		common.HexToHash(self.config.BzzKey), //
		config.HiveParams,                    //
		swapEnabled,                          //
		syncEnabled,                          //
	)
	log.Debug(fmt.Sprintf("Set up swarm network with Kademlia hive"))

//
	self.cloud = network.NewForwarder(self.hive)
	log.Debug(fmt.Sprintf("-> set swarm forwarder as cloud storage backend"))

//
	self.storage = storage.NewNetStore(hash, self.lstore, self.cloud, config.StoreParams)
	log.Debug(fmt.Sprintf("-> swarm net store shared access layer to Swarm Chunk Store"))

//
	self.depo = network.NewDepo(hash, self.lstore, self.storage)
	log.Debug(fmt.Sprintf("-> REmote Access to CHunks"))

//
	dpaChunkStore := storage.NewDpaChunkStore(self.lstore, self.storage)
	log.Debug(fmt.Sprintf("-> Local Access to Swarm"))
//
	self.dpa = storage.NewDPA(dpaChunkStore, self.config.ChunkerParams)
	log.Debug(fmt.Sprintf("-> Content Store API"))

//
	transactOpts := bind.NewKeyedTransactor(self.privateKey)

	if ensClient == nil {
		log.Warn("No ENS, please specify non-empty --ens-api to use domain name resolution")
	} else {
		self.dns, err = ens.NewENS(transactOpts, config.EnsRoot, ensClient)
		if err != nil {
			return nil, err
		}
	}
	log.Debug(fmt.Sprintf("-> Swarm Domain Name Registrar @ address %v", config.EnsRoot.Hex()))

	self.api = api.NewApi(self.dpa, self.dns)
//
	log.Debug(fmt.Sprintf("-> Web3 virtual server API"))

	self.sfs = fuse.NewSwarmFS(self.api)
	log.Debug("-> Initializing Fuse file system")

	return self, nil
}

/*
Start is called when the stack is started
* starts the network kademlia hive peer management
* (starts netStore level 0 api)
* starts DPA level 1 api (chunking -> store/retrieve requests)
* (starts level 2 api)
* starts http proxy server
* registers url scheme handlers for bzz, etc
* TODO: start subservices like sword, swear, swarmdns
*/
//
func (self *Swarm) Start(srv *p2p.Server) error {
	connectPeer := func(url string) error {
		node, err := discover.ParseNode(url)
		if err != nil {
			return fmt.Errorf("invalid node URL: %v", err)
		}
		srv.AddPeer(node)
		return nil
	}
//
	if self.swapEnabled {
		ctx := context.Background() //
		err := self.SetChequebook(ctx)
		if err != nil {
			return fmt.Errorf("Unable to set chequebook for SWAP: %v", err)
		}
		log.Debug(fmt.Sprintf("-> cheque book for SWAP: %v", self.config.Swap.Chequebook()))
	} else {
		log.Debug(fmt.Sprintf("SWAP disabled: no cheque book set"))
	}

	log.Warn(fmt.Sprintf("Starting Swarm service"))
	self.hive.Start(
		discover.PubkeyID(&srv.PrivateKey.PublicKey),
		func() string { return srv.ListenAddr },
		connectPeer,
	)
	log.Info(fmt.Sprintf("Swarm network started on bzz address: %v", self.hive.Addr()))

	self.dpa.Start()
	log.Debug(fmt.Sprintf("Swarm DPA started"))

//
	if self.config.Port != "" {
		addr := net.JoinHostPort(self.config.ListenAddr, self.config.Port)
		go httpapi.StartHttpServer(self.api, &httpapi.ServerConfig{
			Addr:       addr,
			CorsString: self.corsString,
		})
		log.Info(fmt.Sprintf("Swarm http proxy started on %v", addr))

		if self.corsString != "" {
			log.Debug(fmt.Sprintf("Swarm http proxy started with corsdomain: %v", self.corsString))
		}
	}

	return nil
}

//
//
func (self *Swarm) Stop() error {
	self.dpa.Stop()
	self.hive.Stop()
	if ch := self.config.Swap.Chequebook(); ch != nil {
		ch.Stop()
		ch.Save()
	}

	if self.lstore != nil {
		self.lstore.DbStore.Close()
	}
	self.sfs.Stop()
	return self.config.Save()
}

//
func (self *Swarm) Protocols() []p2p.Protocol {
	proto, err := network.Bzz(self.depo, self.backend, self.hive, self.dbAccess, self.config.Swap, self.config.SyncParams, self.config.NetworkId)
	if err != nil {
		return nil
	}
	return []p2p.Protocol{proto}
}

//
//
func (self *Swarm) APIs() []rpc.API {
	return []rpc.API{
//
		{
			Namespace: "bzz",
			Version:   "0.1",
			Service:   &Info{self.config, chequebook.ContractParams},
			Public:    true,
		},
//
		{
			Namespace: "bzz",
			Version:   "0.1",
			Service:   api.NewControl(self.api, self.hive),
			Public:    false,
		},
		{
			Namespace: "chequebook",
			Version:   chequebook.Version,
			Service:   chequebook.NewApi(self.config.Swap.Chequebook),
			Public:    false,
		},
		{
			Namespace: "swarmfs",
			Version:   fuse.Swarmfs_Version,
			Service:   self.sfs,
			Public:    false,
		},
//
//
		{
			Namespace: "bzz",
			Version:   "0.1",
			Service:   api.NewStorage(self.api),
			Public:    true,
		},
		{
			Namespace: "bzz",
			Version:   "0.1",
			Service:   api.NewFileSystem(self.api),
			Public:    false,
		},
//
	}
}

func (self *Swarm) Api() *api.Api {
	return self.api
}

//
func (self *Swarm) SetChequebook(ctx context.Context) error {
	err := self.config.Swap.SetChequebook(ctx, self.backend, self.config.Path)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("new chequebook set (%v): saving config file, resetting all connections in the hive", self.config.Swap.Contract.Hex()))
	self.config.Save()
	self.hive.DropAll()
	return nil
}

//
func NewLocalSwarm(datadir, port string) (self *Swarm, err error) {

	prvKey, err := crypto.GenerateKey()
	if err != nil {
		return
	}

	config, err := api.NewConfig(datadir, common.Address{}, prvKey, network.NetworkId)
	if err != nil {
		return
	}
	config.Port = port

	dpa, err := storage.NewLocalDPA(datadir)
	if err != nil {
		return
	}

	self = &Swarm{
		api:    api.NewApi(dpa, nil),
		config: config,
	}

	return
}

//
type Info struct {
	*api.Config
	*chequebook.Params
}

func (self *Info) Info() *Info {
	return self
}
