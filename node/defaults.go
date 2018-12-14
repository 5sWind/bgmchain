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

package node

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/5sWind/bgmchain/p2p"
	"github.com/5sWind/bgmchain/p2p/nat"
)

const (
	DefaultHTTPHost = "localhost" //
	DefaultHTTPPort = 7575        //
	DefaultWSHost   = "localhost" //
	DefaultWSPort   = 8546        //
)

//
var DefaultConfig = Config{
	DataDir:     DefaultDataDir(),
	HTTPPort:    DefaultHTTPPort,
	HTTPModules: []string{"net", "web3"},
	WSPort:      DefaultWSPort,
	WSModules:   []string{"net", "web3"},
	P2P: p2p.Config{
		ListenAddr:      ":17575",
		DiscoveryV5Addr: ":30304",
		MaxPeers:        25,
		NAT:             nat.Any(),
	},
}

//
//
func DefaultDataDir() string {
//
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Bgmchain")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Bgmchain")
		} else {
			return filepath.Join(home, ".bgmchain")
		}
	}
//
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
