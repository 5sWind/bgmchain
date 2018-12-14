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

package bgmclient

import "github.com/5sWind/bgmchain"

//
var (
	_ = bgmchain.ChainReader(&Client{})
	_ = bgmchain.TransactionReader(&Client{})
	_ = bgmchain.ChainStateReader(&Client{})
	_ = bgmchain.ChainSyncReader(&Client{})
	_ = bgmchain.ContractCaller(&Client{})
	_ = bgmchain.GasEstimator(&Client{})
	_ = bgmchain.GasPricer(&Client{})
	_ = bgmchain.LogFilterer(&Client{})
	_ = bgmchain.PendingStateReader(&Client{})
//
	_ = bgmchain.PendingContractCaller(&Client{})
)
