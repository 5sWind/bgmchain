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

	"github.com/5sWind/bgmchain/log"
)

//
func SetVerbosity(level int) {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(level), log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}
