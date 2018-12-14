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
	"os"
	"runtime"

	"github.com/5sWind/bgmchain/log"
)

func init() {
//
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

//
	runtime.GOMAXPROCS(runtime.NumCPU())
}
