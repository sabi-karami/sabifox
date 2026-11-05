package tunservice

import (
	"os"
	"path/filepath"

	"github.com/sabi-karami/sabifox/core/internal/logx"
)

// The service of the versions before 1.0.9, when the program was called
// SABIFOX Legacy: its unit and its folder. Installing or removing the new
// service removes it.
const (
	legacyUnit   = "sabifox-legacy-tun.service"
	legacyFolder = "/opt/sabifox-legacy-tun"
)

// removeLegacy stops and removes the old service and its folder, if they
// are there; a failure is logged, not returned.
func removeLegacy() {
	file := filepath.Join("/etc/systemd/system", legacyUnit)
	if _, err := os.Stat(file); err == nil {
		_ = run("systemctl", "disable", "--now", legacyUnit)
		if err := os.Remove(file); err != nil {
			logx.Warn("TUN service: removing the old service failed err=%s", logx.Err(err))
		}
		_ = run("systemctl", "daemon-reload")
	}
	if _, err := os.Stat(filepath.Join(legacyFolder, "sabifox-legacy-tun")); err == nil {
		if err := os.RemoveAll(legacyFolder); err != nil {
			logx.Warn("TUN service: removing the old service's folder failed err=%s", logx.Err(err))
		}
	}
}
