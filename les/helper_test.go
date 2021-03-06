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

package les

import (
	"crypto/rand"
	"math/big"
	"sync"
	"testing"

	"github.com/5sWind/bgmchain/common"
	"github.com/5sWind/bgmchain/consensus/bgmash"
	"github.com/5sWind/bgmchain/core"
	"github.com/5sWind/bgmchain/core/types"
	"github.com/5sWind/bgmchain/core/vm"
	"github.com/5sWind/bgmchain/crypto"
	"github.com/5sWind/bgmchain/bgmdb"
	"github.com/5sWind/bgmchain/event"
	"github.com/5sWind/bgmchain/les/flowcontrol"
	"github.com/5sWind/bgmchain/light"
	"github.com/5sWind/bgmchain/p2p"
	"github.com/5sWind/bgmchain/p2p/discover"
	"github.com/5sWind/bgmchain/params"
)

var (
	testBankKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
	testBankFunds   = big.NewInt(1000000000000000000)

	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr   = crypto.PubkeyToAddress(acc2Key.PublicKey)

	testContractCode         = common.Hex2Bytes("606060405260cc8060106000396000f360606040526000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146041578063c16431b914606b57603f565b005b6055600480803590602001909190505060a9565b6040518082815260200191505060405180910390f35b60886004808035906020019091908035906020019091905050608a565b005b80600060005083606481101560025790900160005b50819055505b5050565b6000600060005082606481101560025790900160005b5054905060c7565b91905056")
	testContractAddr         common.Address
	testContractCodeDeployed = testContractCode[16:]
	testContractDeployed     = uint64(2)

	testBufLimit = uint64(100)

	bigTxGas = new(big.Int).SetUint64(params.TxGas)
)

/*
contract test {

    uint256[100] data;

    function Put(uint256 addr, uint256 value) {
        data[addr] = value;
    }

    function Get(uint256 addr) constant returns (uint256 value) {
        return data[addr];
    }
}
*/

func testChainGen(i int, block *core.BlockGen) {
	signer := types.HomesteadSigner{}

	switch i {
	case 0:
//
		tx, _ := types.SignTx(types.NewTransaction(types.Binary, block.TxNonce(testBankAddress), acc1Addr, big.NewInt(10000), bigTxGas, nil, nil), signer, testBankKey)
		block.AddTx(tx)
	case 1:
//
//
//
		tx1, _ := types.SignTx(types.NewTransaction(types.Binary, block.TxNonce(testBankAddress), acc1Addr, big.NewInt(1000), bigTxGas, nil, nil), signer, testBankKey)
		nonce := block.TxNonce(acc1Addr)
		tx2, _ := types.SignTx(types.NewTransaction(types.Binary, nonce, acc2Addr, big.NewInt(1000), bigTxGas, nil, nil), signer, acc1Key)
		nonce++
		tx3, _ := types.SignTx(types.NewContractCreation(nonce, big.NewInt(0), big.NewInt(200000), big.NewInt(0), testContractCode), signer, acc1Key)
		testContractAddr = crypto.CreateAddress(acc1Addr, nonce)
		block.AddTx(tx1)
		block.AddTx(tx2)
		block.AddTx(tx3)
	case 2:
//
		block.SetCoinbase(acc2Addr)
		block.SetExtra([]byte("yeehaw"))
		data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001")
		tx, _ := types.SignTx(types.NewTransaction(types.Binary, block.TxNonce(testBankAddress), testContractAddr, big.NewInt(0), big.NewInt(100000), nil, data), signer, testBankKey)
		block.AddTx(tx)
	case 3:
//
		b2 := block.PrevBlock(1).Header()
		b2.Extra = []byte("foo")
		block.AddUncle(b2)
		b3 := block.PrevBlock(2).Header()
		b3.Extra = []byte("foo")
		block.AddUncle(b3)
		data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000002")
		tx, _ := types.SignTx(types.NewTransaction(types.Binary, block.TxNonce(testBankAddress), testContractAddr, big.NewInt(0), big.NewInt(100000), nil, data), signer, testBankKey)
		block.AddTx(tx)
	}
}

func testRCL() RequestCostList {
	cl := make(RequestCostList, len(reqList))
	for i, code := range reqList {
		cl[i].MsgCode = code
		cl[i].BaseCost = 0
		cl[i].ReqCost = 0
	}
	return cl
}

//
//
//
func newTestProtocolManager(lightSync bool, blocks int, generator func(int, *core.BlockGen), peers *peerSet, odr *LesOdr, db bgmdb.Database) (*ProtocolManager, error) {
	var (
		evmux  = new(event.TypeMux)
		engine = bgmash.NewFaker()
		gspec  = core.Genesis{
			Config: params.TestChainConfig,
			Alloc:  core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
		}
		genesis = gspec.MustCommit(db)
		chain   BlockChain
	)
	if peers == nil {
		peers = newPeerSet()
	}

	if lightSync {
		chain, _ = light.NewLightChain(odr, gspec.Config, engine)
	} else {
		blockchain, _ := core.NewBlockChain(db, gspec.Config, engine, vm.Config{})
		gchain, _ := core.GenerateChain(gspec.Config, genesis, db, blocks, generator)
		if _, err := blockchain.InsertChain(gchain); err != nil {
			panic(err)
		}
		chain = blockchain
	}

	var protocolVersions []uint
	if lightSync {
		protocolVersions = ClientProtocolVersions
	} else {
		protocolVersions = ServerProtocolVersions
	}
	pm, err := NewProtocolManager(gspec.Config, lightSync, protocolVersions, NetworkId, evmux, engine, peers, chain, nil, db, odr, nil, make(chan struct{}), new(sync.WaitGroup))
	if err != nil {
		return nil, err
	}
	if !lightSync {
		srv := &LesServer{protocolManager: pm}
		pm.server = srv

		srv.defParams = &flowcontrol.ServerParams{
			BufLimit:    testBufLimit,
			MinRecharge: 1,
		}

		srv.fcManager = flowcontrol.NewClientManager(50, 10, 1000000000)
		srv.fcCostStats = newCostStats(nil)
	}
	pm.Start()
	return pm, nil
}

//
//
//
//
func newTestProtocolManagerMust(t *testing.T, lightSync bool, blocks int, generator func(int, *core.BlockGen), peers *peerSet, odr *LesOdr, db bgmdb.Database) *ProtocolManager {
	pm, err := newTestProtocolManager(lightSync, blocks, generator, peers, odr, db)
	if err != nil {
		t.Fatalf("Failed to create protocol manager: %v", err)
	}
	return pm
}

//
type testPeer struct {
	net p2p.MsgReadWriter //
	app *p2p.MsgPipeRW    //
	*peer
}

//
func newTestPeer(t *testing.T, name string, version int, pm *ProtocolManager, shake bool) (*testPeer, <-chan error) {
//
	app, net := p2p.MsgPipe()

//
	var id discover.NodeID
	rand.Read(id[:])

	peer := pm.newPeer(version, NetworkId, p2p.NewPeer(id, name, nil), net)

//
	errc := make(chan error, 1)
	go func() {
		select {
		case pm.newPeerCh <- peer:
			errc <- pm.handle(peer)
		case <-pm.quitSync:
			errc <- p2p.DiscQuitting
		}
	}()
	tp := &testPeer{
		app:  app,
		net:  net,
		peer: peer,
	}
//
	if shake {
		td, head, genesis := pm.blockchain.Status()
		headNum := pm.blockchain.CurrentHeader().Number.Uint64()
		tp.handshake(t, td, head, headNum, genesis)
	}
	return tp, errc
}

func newTestPeerPair(name string, version int, pm, pm2 *ProtocolManager) (*peer, <-chan error, *peer, <-chan error) {
//
	app, net := p2p.MsgPipe()

//
	var id discover.NodeID
	rand.Read(id[:])

	peer := pm.newPeer(version, NetworkId, p2p.NewPeer(id, name, nil), net)
	peer2 := pm2.newPeer(version, NetworkId, p2p.NewPeer(id, name, nil), app)

//
	errc := make(chan error, 1)
	errc2 := make(chan error, 1)
	go func() {
		select {
		case pm.newPeerCh <- peer:
			errc <- pm.handle(peer)
		case <-pm.quitSync:
			errc <- p2p.DiscQuitting
		}
	}()
	go func() {
		select {
		case pm2.newPeerCh <- peer2:
			errc2 <- pm2.handle(peer2)
		case <-pm2.quitSync:
			errc2 <- p2p.DiscQuitting
		}
	}()
	return peer, errc, peer2, errc2
}

//
//
func (p *testPeer) handshake(t *testing.T, td *big.Int, head common.Hash, headNum uint64, genesis common.Hash) {
	var expList keyValueList
	expList = expList.add("protocolVersion", uint64(p.version))
	expList = expList.add("networkId", uint64(NetworkId))
	expList = expList.add("headTd", td)
	expList = expList.add("headHash", head)
	expList = expList.add("headNum", headNum)
	expList = expList.add("genesisHash", genesis)
	sendList := make(keyValueList, len(expList))
	copy(sendList, expList)
	expList = expList.add("serveHeaders", nil)
	expList = expList.add("serveChainSince", uint64(0))
	expList = expList.add("serveStateSince", uint64(0))
	expList = expList.add("txRelay", nil)
	expList = expList.add("flowControl/BL", testBufLimit)
	expList = expList.add("flowControl/MRR", uint64(1))
	expList = expList.add("flowControl/MRC", testRCL())

	if err := p2p.ExpectMsg(p.app, StatusMsg, expList); err != nil {
		t.Fatalf("status recv: %v", err)
	}
	if err := p2p.Send(p.app, StatusMsg, sendList); err != nil {
		t.Fatalf("status send: %v", err)
	}

	p.fcServerParams = &flowcontrol.ServerParams{
		BufLimit:    testBufLimit,
		MinRecharge: 1,
	}
}

//
//
func (p *testPeer) close() {
	p.app.Close()
}
