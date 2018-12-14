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
	"github.com/5sWind/bgmchain/metrics"
	"github.com/5sWind/bgmchain/p2p"
)

var (
	/*	propTxnInPacketsMeter     = metrics.NewMeter("bgm/prop/txns/in/packets")
		propTxnInTrafficMeter     = metrics.NewMeter("bgm/prop/txns/in/traffic")
		propTxnOutPacketsMeter    = metrics.NewMeter("bgm/prop/txns/out/packets")
		propTxnOutTrafficMeter    = metrics.NewMeter("bgm/prop/txns/out/traffic")
		propHashInPacketsMeter    = metrics.NewMeter("bgm/prop/hashes/in/packets")
		propHashInTrafficMeter    = metrics.NewMeter("bgm/prop/hashes/in/traffic")
		propHashOutPacketsMeter   = metrics.NewMeter("bgm/prop/hashes/out/packets")
		propHashOutTrafficMeter   = metrics.NewMeter("bgm/prop/hashes/out/traffic")
		propBlockInPacketsMeter   = metrics.NewMeter("bgm/prop/blocks/in/packets")
		propBlockInTrafficMeter   = metrics.NewMeter("bgm/prop/blocks/in/traffic")
		propBlockOutPacketsMeter  = metrics.NewMeter("bgm/prop/blocks/out/packets")
		propBlockOutTrafficMeter  = metrics.NewMeter("bgm/prop/blocks/out/traffic")
		reqHashInPacketsMeter     = metrics.NewMeter("bgm/req/hashes/in/packets")
		reqHashInTrafficMeter     = metrics.NewMeter("bgm/req/hashes/in/traffic")
		reqHashOutPacketsMeter    = metrics.NewMeter("bgm/req/hashes/out/packets")
		reqHashOutTrafficMeter    = metrics.NewMeter("bgm/req/hashes/out/traffic")
		reqBlockInPacketsMeter    = metrics.NewMeter("bgm/req/blocks/in/packets")
		reqBlockInTrafficMeter    = metrics.NewMeter("bgm/req/blocks/in/traffic")
		reqBlockOutPacketsMeter   = metrics.NewMeter("bgm/req/blocks/out/packets")
		reqBlockOutTrafficMeter   = metrics.NewMeter("bgm/req/blocks/out/traffic")
		reqHeaderInPacketsMeter   = metrics.NewMeter("bgm/req/headers/in/packets")
		reqHeaderInTrafficMeter   = metrics.NewMeter("bgm/req/headers/in/traffic")
		reqHeaderOutPacketsMeter  = metrics.NewMeter("bgm/req/headers/out/packets")
		reqHeaderOutTrafficMeter  = metrics.NewMeter("bgm/req/headers/out/traffic")
		reqBodyInPacketsMeter     = metrics.NewMeter("bgm/req/bodies/in/packets")
		reqBodyInTrafficMeter     = metrics.NewMeter("bgm/req/bodies/in/traffic")
		reqBodyOutPacketsMeter    = metrics.NewMeter("bgm/req/bodies/out/packets")
		reqBodyOutTrafficMeter    = metrics.NewMeter("bgm/req/bodies/out/traffic")
		reqStateInPacketsMeter    = metrics.NewMeter("bgm/req/states/in/packets")
		reqStateInTrafficMeter    = metrics.NewMeter("bgm/req/states/in/traffic")
		reqStateOutPacketsMeter   = metrics.NewMeter("bgm/req/states/out/packets")
		reqStateOutTrafficMeter   = metrics.NewMeter("bgm/req/states/out/traffic")
		reqReceiptInPacketsMeter  = metrics.NewMeter("bgm/req/receipts/in/packets")
		reqReceiptInTrafficMeter  = metrics.NewMeter("bgm/req/receipts/in/traffic")
		reqReceiptOutPacketsMeter = metrics.NewMeter("bgm/req/receipts/out/packets")
		reqReceiptOutTrafficMeter = metrics.NewMeter("bgm/req/receipts/out/traffic")*/
	miscInPacketsMeter  = metrics.NewMeter("les/misc/in/packets")
	miscInTrafficMeter  = metrics.NewMeter("les/misc/in/traffic")
	miscOutPacketsMeter = metrics.NewMeter("les/misc/out/packets")
	miscOutTrafficMeter = metrics.NewMeter("les/misc/out/traffic")
)

//
//
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     //
	version           int //
}

//
//
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

//
//
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
//
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
//
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
//
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

//
	return rw.MsgReadWriter.WriteMsg(msg)
}
