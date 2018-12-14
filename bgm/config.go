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
	"math/big"
	"os"
	"os/user"

	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/common/hexutil"
	"github.com/5sWind/bgmchain/core"
	"github.com/5sWind/bgmchain/bgm/downloader"
	"github.com/5sWind/bgmchain/bgm/gasprice"
	"github.com/5sWind/bgmchain/params"
)

//
var DefaultConfig = Config{
	SyncMode:      downloader.FullSync,
	NetworkId:     1357,
	LightPeers:    20,
	DatabaseCache: 128,
	GasPrice:      big.NewInt(18 * params.Shannon),

	TxPool: core.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     10,
		Percentile: 50,
	},
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
}

//go:generate gencodec -type Config -field-override configMarshaling -formats toml -out gen_config.go

type Config struct {
//
//
	Genesis *core.Genesis `toml:",omitempty"`

//
	NetworkId uint64 //
	SyncMode  downloader.SyncMode

//
	LightServ  int `toml:",omitempty"` //
	LightPeers int `toml:",omitempty"` //

//
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int

//
	Validator    common.Address `toml:",omitempty"`
	Coinbase     common.Address `toml:",omitempty"`
	MinerThreads int            `toml:",omitempty"`
	ExtraData    []byte         `toml:",omitempty"`
	GasPrice     *big.Int

//
	TxPool core.TxPoolConfig

//
	GPO gasprice.Config

//
	EnablePreimageRecording bool

//
	DocRoot   string `toml:"-"`
	PowFake   bool   `toml:"-"`
	PowTest   bool   `toml:"-"`
	PowShared bool   `toml:"-"`
	Dpos      bool   `toml:"-"`
}

type configMarshaling struct {
	ExtraData hexutil.Bytes
}
