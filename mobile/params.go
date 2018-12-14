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
	"github.com/5sWind/bgmchain/p2p/discv5"
	"github.com/5sWind/bgmchain/params"
)

//
//
func MainnetGenesis() string {
	return ""
}

//
//
func FoundationBootnodes() *Enodes {
	nodes := &Enodes{nodes: make([]*discv5.Node, len(params.DiscoveryV5Bootnodes))}
	for i, url := range params.DiscoveryV5Bootnodes {
		nodes.nodes[i] = discv5.MustParseNode(url)
	}
	return nodes
}
