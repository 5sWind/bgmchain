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
//

package gbgm

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/5sWind/bgmchain/core"
	"github.com/5sWind/bgmchain/bgm"
	"github.com/5sWind/bgmchain/bgm/downloader"
	"github.com/5sWind/bgmchain/bgmclient"
	"github.com/5sWind/bgmchain/bgmstats"
	"github.com/5sWind/bgmchain/les"
	"github.com/5sWind/bgmchain/node"
	"github.com/5sWind/bgmchain/p2p"
	"github.com/5sWind/bgmchain/p2p/nat"
	"github.com/5sWind/bgmchain/params"
	whisper "github.com/5sWind/bgmchain/whisper/whisperv5"
)

//
//
//
//
type NodeConfig struct {
//
	BootstrapNodes *Enodes

//
//
	MaxPeers int

//
	BgmchainEnabled bool

//
//
	BgmchainNetworkID int64 //

//
//
	BgmchainGenesis string

//
//
	BgmchainDatabaseCache int

//
//
	//
//
	BgmchainNetStats string

//
	WhisperEnabled bool
}

//
//
var defaultNodeConfig = &NodeConfig{
	BootstrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	BgmchainEnabled:       true,
	BgmchainNetworkID:     1,
	BgmchainDatabaseCache: 16,
}

//
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

//
type Node struct {
	node *node.Node
}

//
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
//
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BootstrapNodes == nil || config.BootstrapNodes.Size() == 0 {
		config.BootstrapNodes = defaultNodeConfig.BootstrapNodes
	}
//
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.Version,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), //
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			DiscoveryV5Addr:  ":0",
			BootstrapNodesV5: config.BootstrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	var genesis *core.Genesis
	if config.BgmchainGenesis != "" {
//
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.BgmchainGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
	}
//
	if config.BgmchainEnabled {
		bgmConf := bgm.DefaultConfig
		bgmConf.Genesis = genesis
		bgmConf.SyncMode = downloader.LightSync
		bgmConf.NetworkId = uint64(config.BgmchainNetworkID)
		bgmConf.DatabaseCache = config.BgmchainDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &bgmConf)
		}); err != nil {
			return nil, fmt.Errorf("bgmchain init: %v", err)
		}
//
		if config.BgmchainNetStats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.LightBgmchain
				ctx.Service(&lesServ)

				return bgmstats.New(config.BgmchainNetStats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("netstats init: %v", err)
			}
		}
	}
//
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) {
			return whisper.New(&whisper.DefaultConfig), nil
		}); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

//
func (n *Node) Start() error {
	return n.node.Start()
}

//
//
func (n *Node) Stop() error {
	return n.node.Stop()
}

//
func (n *Node) GetBgmchainClient() (client *BgmchainClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &BgmchainClient{bgmclient.NewClient(rpc)}, nil
}

//
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

//
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
