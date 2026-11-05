package tunservice

import (
	"os"
	"path/filepath"

	"github.com/sabi-karami/sabifox/core/internal/logx"
)

// The folder of the service of the versions before 1.0.9, when the program
// was called SABIFOX Legacy. Its launchd job has the same label as the new
// one's, so installing the new service stops and replaces it; the folder
// goes then, and when the new service is removed.
const legacyFolder = "/Library/Application Support/SABIFOX Legacy TUN"

// removeLegacy removes the old service's folder, if it is there; a failure
// is logged, not returned.
func removeLegacy() {
	if _, err := os.Stat(filepath.Join(legacyFolder, "sabifox-legacy-tun")); err == nil {
		if err := os.RemoveAll(legacyFolder); err != nil {
			logx.Warn("TUN service: removing the old service's folder failed err=%s", logx.Err(err))
		}
	}
}
