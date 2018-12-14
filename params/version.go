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

package params

import (
	"fmt"
)

const (
	VersionMajor = 1        //
	VersionMinor = 7        //
	VersionPatch = 4        //
	VersionMeta  = "stable" //
)

//
var Version = func() string {
	v := fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

func VersionWithCommit(gitCommit string) string {
	vsn := Version
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}
